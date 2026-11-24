//! Bounded FIFO circular buffer implementation.

use std::fmt;

/// Strategy to apply when pushing to a full buffer.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OverflowPolicy {
    /// Reject the new element and return an error.
    Reject,
    /// Overwrite the oldest element in the buffer.
    Overwrite,
}

/// Errors that can occur during buffer operations.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RingBufferError {
    /// Buffer is at maximum capacity under `OverflowPolicy::Reject`.
    BufferFull,
    /// Buffer is empty when attempting to pop.
    BufferEmpty,
    /// Capacity supplied during initialization was zero.
    ZeroCapacity,
}

impl fmt::Display for RingBufferError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            RingBufferError::BufferFull => write!(f, "Ring buffer is full"),
            RingBufferError::BufferEmpty => write!(f, "Ring buffer is empty"),
            RingBufferError::ZeroCapacity => write!(f, "Ring buffer capacity must be greater than zero"),
        }
    }
}

impl std::error::Error for RingBufferError {}

/// A bounded circular buffer storing elements of type `T`.
#[derive(Debug, Clone)]
pub struct RingBuffer<T> {
    storage: Vec<Option<T>>,
    capacity: usize,
    head: usize,
    tail: usize,
    size: usize,
    overflow_policy: OverflowPolicy,
}

impl<T> RingBuffer<T> {
    /// Creates a new `RingBuffer` with the specified capacity and overflow policy.
    pub fn new(capacity: usize, overflow_policy: OverflowPolicy) -> Result<Self, RingBufferError> {
        if capacity == 0 {
            return Err(RingBufferError::ZeroCapacity);
        }

        let mut storage = Vec::with_capacity(capacity);
        for _ in 0..capacity {
            storage.push(None);
        }

        Ok(Self {
            storage,
            capacity,
            head: 0,
            tail: 0,
            size: 0,
            overflow_policy,
        })
    }

    /// Returns the maximum capacity of the buffer.
    pub fn capacity(&self) -> usize {
        self.capacity
    }

    /// Returns the number of elements currently stored in the buffer.
    pub fn len(&self) -> usize {
        self.size
    }

    /// Returns `true` if the buffer contains no elements.
    pub fn is_empty(&self) -> bool {
        self.size == 0
    }

    /// Returns `true` if the buffer has reached maximum capacity.
    pub fn is_full(&self) -> bool {
        self.size == self.capacity
    }

    /// Pushes an item into the buffer according to the configured `OverflowPolicy`.
    pub fn push(&mut self, item: T) -> Result<(), RingBufferError> {
        if self.is_full() {
            match self.overflow_policy {
                OverflowPolicy::Reject => return Err(RingBufferError::BufferFull),
                OverflowPolicy::Overwrite => {
                    // Overwrite oldest item at head and advance head
                    self.storage[self.head] = Some(item);
                    self.head = (self.head + 1) % self.capacity;
                    self.tail = (self.tail + 1) % self.capacity;
                    return Ok(());
                }
            }
        }

        self.storage[self.tail] = Some(item);
        self.tail = (self.tail + 1) % self.capacity;
        self.size += 1;
        Ok(())
    }

    /// Removes and returns the oldest element from the buffer.
    pub fn pop(&mut self) -> Result<T, RingBufferError> {
        if self.is_empty() {
            return Err(RingBufferError::BufferEmpty);
        }

        let item = self.storage[self.head].take().ok_or(RingBufferError::BufferEmpty)?;
        self.head = (self.head + 1) % self.capacity;
        self.size -= 1;
        Ok(item)
    }

    /// Returns a reference to the oldest element without removing it.
    pub fn peek(&self) -> Option<&T> {
        if self.is_empty() {
            None
        } else {
            self.storage[self.head].as_ref()
        }
    }

    /// Clears all elements from the buffer.
    pub fn clear(&mut self) {
        for slot in self.storage.iter_mut() {
            *slot = None;
        }
        self.head = 0;
        self.tail = 0;
        self.size = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zero_capacity_rejected() {
        assert_eq!(
            RingBuffer::<i32>::new(0, OverflowPolicy::Reject).unwrap_err(),
            RingBufferError::ZeroCapacity
        );
    }

    #[test]
    fn test_fifo_ordering() {
        let mut buf = RingBuffer::new(3, OverflowPolicy::Reject).unwrap();
        assert!(buf.is_empty());
        assert_eq!(buf.len(), 0);

        buf.push(10).unwrap();
        buf.push(20).unwrap();
        buf.push(30).unwrap();

        assert!(buf.is_full());
        assert_eq!(buf.len(), 3);
        assert_eq!(buf.peek(), Some(&10));

        assert_eq!(buf.pop().unwrap(), 10);
        assert_eq!(buf.pop().unwrap(), 20);
        assert_eq!(buf.pop().unwrap(), 30);
        assert_eq!(buf.pop().unwrap_err(), RingBufferError::BufferEmpty);
    }

    #[test]
    fn test_overflow_reject() {
        let mut buf = RingBuffer::new(2, OverflowPolicy::Reject).unwrap();
        buf.push(1).unwrap();
        buf.push(2).unwrap();
        assert_eq!(buf.push(3).unwrap_err(), RingBufferError::BufferFull);
        assert_eq!(buf.len(), 2);
    }

    #[test]
    fn test_overflow_overwrite() {
        let mut buf = RingBuffer::new(2, OverflowPolicy::Overwrite).unwrap();
        buf.push(100).unwrap();
        buf.push(200).unwrap();

        // Overwrite 100 with 300
        buf.push(300).unwrap();
        assert_eq!(buf.len(), 2);
        assert_eq!(buf.pop().unwrap(), 200);
        assert_eq!(buf.pop().unwrap(), 300);
    }

    #[test]
    fn test_clear_and_reuse() {
        let mut buf = RingBuffer::new(3, OverflowPolicy::Reject).unwrap();
        buf.push(1).unwrap();
        buf.push(2).unwrap();
        buf.clear();

        assert!(buf.is_empty());
        assert_eq!(buf.len(), 0);
        assert_eq!(buf.peek(), None);

        buf.push(42).unwrap();
        assert_eq!(buf.pop().unwrap(), 42);
    }
}
