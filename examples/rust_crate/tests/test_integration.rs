//! Integration tests for read-ringbuffer crate.

use read_ringbuffer::{OverflowPolicy, RingBuffer, RingBufferError};

#[test]
fn test_ring_buffer_concurrency_simulation() {
    let mut buffer = RingBuffer::new(50, OverflowPolicy::Reject).expect("Valid capacity");

    for i in 0..50 {
        buffer.push(format!("event-{}", i)).expect("Push within capacity");
    }

    assert!(buffer.is_full());
    assert_eq!(buffer.push("overflow-event".to_string()), Err(RingBufferError::BufferFull));

    for i in 0..50 {
        let val = buffer.pop().expect("Pop item");
        assert_eq!(val, format!("event-{}", i));
    }

    assert!(buffer.is_empty());
}

#[test]
fn test_sliding_window_overwrite_stream() {
    let mut buffer = RingBuffer::new(3, OverflowPolicy::Overwrite).expect("Valid capacity");

    for i in 1..=10 {
        buffer.push(i).expect("Overwrite never errors");
    }

    assert_eq!(buffer.len(), 3);
    assert_eq!(buffer.pop().unwrap(), 8);
    assert_eq!(buffer.pop().unwrap(), 9);
    assert_eq!(buffer.pop().unwrap(), 10);
}
