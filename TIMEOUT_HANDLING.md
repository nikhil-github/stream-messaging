# Timeout Handling Strategy

## Problem Statement

When consuming messages from a broker, timeouts can occur in two scenarios:
1. **No messages available** - The consumer waited but no messages arrived (this is NORMAL)
2. **Real errors** - Connection issues, consumer closed, etc. (this is an ERROR)

Traditional APIs treat both as errors, forcing developers to write complex error handling:

```go
// ❌ Bad: Timeout treated as error
messages, err := consumer.PullBatch(ctx)
if err != nil {
    if err == nats.ErrTimeout {
        // Not really an error, just no messages...
        return
    }
    // Real error
    log.Fatal(err)
}
```

## Our Solution

We distinguish between "no messages" and "real errors" by handling timeouts gracefully:

```go
// ✅ Good: Timeout returns empty slice, not error
messages, err := consumer.PullBatch(ctx)
if err != nil {
    // This is a REAL error only
    log.Fatal(err)
}

if len(messages) == 0 {
    // No messages available - normal, not an error
    log.Println("No messages, will retry...")
}
```

## Implementation

### In `broker/consumer.go`

```go
func (c *jsConsumer) PullBatch(ctx context.Context) ([]*Message, error) {
    msgs, err := c.sub.Fetch(c.batchSize, nats.Context(ctx))
    if err != nil {
        // Handle context cancellation/timeout
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
        
        // NATS timeout is not an error - it means no messages available
        if err == nats.ErrTimeout {
            return []*Message{}, nil  // ✅ Empty slice, no error
        }
        
        // Other errors are real errors
        return nil, err
    }
    
    // Convert and return messages
    out := make([]*Message, 0, len(msgs))
    for _, m := range msgs {
        out = append(out, c.convertMessage(m))
    }
    return out, nil
}
```

### Error Handling Matrix

| Scenario | Return Value | Behavior |
|----------|--------------|----------|
| **Messages available** | `([]*Message{...}, nil)` | Normal - process messages |
| **No messages (timeout)** | `([]*Message{}, nil)` | Normal - empty slice, no error |
| **Context cancelled** | `(nil, context.Canceled)` | Error - caller cancelled |
| **Context deadline exceeded** | `(nil, context.DeadlineExceeded)` | Error - timeout exceeded |
| **Connection error** | `(nil, error)` | Error - real failure |
| **Consumer closed** | `(nil, ErrConsumerClosed)` | Error - invalid state |

## Benefits

### 1. Cleaner Application Code

**Before:**
```go
for {
    messages, err := consumer.PullBatch(ctx)
    if err != nil {
        if err == nats.ErrTimeout {
            continue  // Not really an error
        }
        log.Fatal(err)
    }
    // Process messages
}
```

**After:**
```go
for {
    messages, err := consumer.PullBatch(ctx)
    if err != nil {
        log.Fatal(err)  // Only real errors reach here
    }
    
    for _, msg := range messages {
        // Process (empty slice is fine, just skips)
        process(msg)
    }
}
```

### 2. Better Semantics

- ✅ **Empty result** = No data available (normal)
- ✅ **Error** = Something went wrong (exceptional)

This follows the principle: *Errors should be exceptional, not expected*

### 3. Easier Testing

```go
func TestNoMessagesAvailable(t *testing.T) {
    messages, err := consumer.PullBatch(ctx)
    
    // No error on timeout
    assert.NoError(t, err)
    
    // Empty slice returned
    assert.Empty(t, messages)
}
```

### 4. Consistent with Other APIs

Similar to:
- `io.EOF` is often handled specially, not as an error
- `sql.Rows.Next()` returns false when no more rows (not an error)
- HTTP 404 vs 500 - "not found" vs "server error"

## Example: Polling Loop

### Simple Polling

```go
func pollMessages(ctx context.Context, consumer broker.Consumer) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            messages, err := consumer.PullBatch(ctx)
            if err != nil {
                // Real error - log and maybe reconnect
                log.Error("Failed to pull messages", err)
                continue
            }
            
            if len(messages) == 0 {
                log.Debug("No messages available")
                continue
            }
            
            // Process messages
            for _, msg := range messages {
                process(msg)
                msg.Ack()
            }
        }
    }
}
```

### Continuous Polling (No Ticker)

```go
func continuousPoll(ctx context.Context, consumer broker.Consumer) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            messages, err := consumer.PullBatch(ctx)
            if err != nil {
                log.Error("Failed to pull", err)
                time.Sleep(5 * time.Second)  // Backoff on error
                continue
            }
            
            // Process whatever we got (including zero messages)
            for _, msg := range messages {
                process(msg)
                msg.Ack()
            }
            
            // If no messages, the fetch already waited, so no sleep needed
        }
    }
}
```

## Context Timeout Recommendations

When using context timeouts, consider:

```go
// Short timeout - fast feedback when no messages
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
messages, err := consumer.PullBatch(ctx)
cancel()

if err == context.DeadlineExceeded {
    // Timeout exceeded - no messages in 1 second
    // This IS an error (different from NATS timeout)
} else if err != nil {
    // Other error
} else if len(messages) == 0 {
    // NATS timeout (happened before context timeout)
}
```

**Best practice:** Let NATS handle the timeout, don't use aggressive context deadlines:

```go
// ✅ Good: No artificial deadline, let NATS timeout naturally
messages, err := consumer.PullBatch(context.Background())
```

Or use context for cancellation only:
```go
// ✅ Good: Cancel when shutting down, but no deadline
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

messages, err := consumer.PullBatch(ctx)
```

## Subscribe Method (Channel-Based)

The `Subscribe()` method already handles timeouts internally:

```go
func (c *jsConsumer) Subscribe(ctx context.Context) (<-chan *Message, error) {
    msgChan := make(chan *Message, c.batchSize)
    
    go func() {
        defer close(msgChan)
        for {
            msgs, err := c.sub.Fetch(c.batchSize, nats.MaxWait(5*time.Second))
            if err != nil {
                if err == nats.ErrTimeout {
                    continue  // No messages, try again
                }
                return  // Real error, stop
            }
            
            for _, msg := range msgs {
                msgChan <- c.convertMessage(msg)
            }
        }
    }()
    
    return msgChan, nil
}
```

Users don't see timeouts at all:
```go
msgChan, _ := consumer.Subscribe(ctx)
for msg := range msgChan {
    // Just process messages, timeouts are hidden
    process(msg)
}
```

## Summary

✅ **NATS timeout** (no messages) → Return empty slice, no error  
✅ **Context timeout** → Return `context.DeadlineExceeded` error  
✅ **Context cancelled** → Return `context.Canceled` error  
✅ **Real errors** → Return error  

This makes the API:
- **Easier to use** - no timeout special-casing needed
- **More intuitive** - empty result ≠ error
- **Better UX** - polling loops are cleaner
- **Consistent** - follows Go best practices

Your application code becomes simpler and more robust! 🎉

