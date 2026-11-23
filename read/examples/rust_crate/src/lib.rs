//! High-performance bounded circular ring buffer library.
//!
//! Provides deterministic FIFO storage with explicit overflow policies.

pub mod ring_buffer;

pub use ring_buffer::{OverflowPolicy, RingBuffer, RingBufferError};
