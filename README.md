

## Requirements

- The connection can be dropped at any time, it should be possible to re-established the connection and complete the stream of numbers (from the current point in the sequence).
- Server provides a series of `n` random numbers (uint32) to the client, `n` is specified by the client when it connects. 
- The client outputs the final checksum(s) and an indication of success or failure.
- The server ticks every second, on each tick the server can send a message.


## Message format

This protocol is [TLV (type-length-value)](https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value) encoded data stream in binary format.

`Connect`

    `Byte3('CON')`
        Identifies the message as a new connection

    `Int32`
        Number of numbers to receive

    `Int32`
        Length of the message content in bytes, not including self

    `String`
        Client ID

Connect can be used to both initialise a new connection, and to resume a dropped connection. 
The connect message contains the `n` number of numbers to receive in sequence, and the client id. 
If the client id is unknown to the server, it's a new connection, if the client id is known to the server
then the sequence should be resumed from the last acknowledged message. 

`Number`

    `Byte3('NUM')`
        Identifies the message as a number

    `Int32`
        Sequence number of this number in the series

    `Int32`
       Length of the number in bytes, not including self

    `ByteN`
       The payload, L length long, where L is specified by the previous length field. This encodes the number as BigEndian

Number is a single number in the sequence. The Number message contains both the number field, and a sequence number. 
The sequence number is incrementing for each number in the sequence. First message 1, second message 2, etc. 
The client should expect it could receive the same number in the sequence multiple times.

`Acknowledge`

    `Byte3('ACK')`
        Identifies the message as a sequence acknowledgement

    `Int32`
        Sequence number to acknowledge

Acknowledge message acknowledges an element in the series of numbers.
The server **must not** wait for an acknowledgement on the previous message before sending the next message. 
The client should acknowledge the numbers in the sequence. 
The server should expect it could receive an acknowledgement for the same sequence number multiple times.
The server assumes that messages are undelivered until it receives an Acknowledge for that messages sequence number.
The server should continue with the sequence of numbers. Once the sequence has completed, the server should use the remaining ticks to send the unacknowledged messages before
sending the Checksum to mark the end of transmission.

`Checksum`

    `Byte3('CHK')`
        Identifies the message as a Checksum

    `Int32`
        Sequence number of this number in the series

    `Int32`
        Length of the checksum data in bytes, not including self

    `String`
        Checksum data

Checksum message indicates the end of the sequence, and carries a checksum value. 
The checksum is [....]
The client must acknowledge the checksum message. An acknowledgement of the checksum message indicates to the server that it's job is done. 
The server should not attempt to send the checksum message until it has received an acknowledgement for all numbers in the series.
If the server exhausts it's series of numbers, and is missing acknowledgements for some numbers in the series, it should use it's next server-ticks to resend those numbers.
If the server has received acknowledgements for all the numbers in the series, but is missing the acknowledgement for the checksum, the server should resend the checksum until the
acknowledgement is received.

## Open questions

- One of the requirements reads: _"The timing of the messages should be under control of the server (that is: information may be sent by the client as part of initiating a
  connection, but there is no per-message client request)."_
  The first part is clear, the messages should be sent or triggered to send by the server. So the protocol should not be a request/response protocol driven by the client. i.e. we
  should not use semantics like `GET /number/{sequence}` to trigger the next send from the server. BUT; does this mean we can't ACK individual messages? I've assumed there that as
  long as the ACK of the previous message is not _triggering_ the send of the next message, then it's fine. If my assumption does not satisfy the requirement, then I would remove
  the `Acknowledge` message type, and add a new message type for `Missing`

    `Missing`
    
        `Byte3('MIS')`
            Identifies the message as Missing
    
        `Int32`
            The number of int32 sequence numbers carried in this message
    
        `Int32...`
            Repeated Int32 sequence numbers that are missing. There are N of these numbers, where N is specified by the previous field.

  Using this missing message, the client can indicate to the server which sequence numbers are missing.
  If the client has not received any message in 2 seconds, then it can send a `Missing` message to the server. 
  Sequence numbers increase by exactly 1 each time. So the client can tell if there's a missing sequence number in the middle of the series. 
  The `Checksum` message marks the end of the series. If the client has received a checksum message then it knows the end of the sequence numbers and can find the missing sequence
  numbers. If the client has not received a checksum message, then it can find the missing sequence numbers that it already knows about AND it can include the highest sequence
  number +1. This is safe because the client knows that it must receive at least one more message, up until the checksum message is received. The draw back here is that for `S`
  missing messages, the client will have to wait at least `S * [wait time]`, where the wait time is currently 2 seconds in order to indicate one by one to the server that
  messages are missing. If the client sends a `Missing` message, and does not receive any messages in return, after 2 seconds, the client should resend the `Missing` message.

  Using this pattern, the server can assume that all messages are successfully delivered. On receiving a `Missing` message, the server can mark individual messages as undelivered,
  and send those again on each server tick. 

  Notes:
  - I've tried to comply with the spirit of the task, but because of the added wait time on the `Missing` message, I've opted to use per-message acknowledgements instead of
    a missing message. This makes the protocol faster in the worst case, as the server receives feedback more quickly that messages are undelivered.
  - This missing message could be flipped around to be a multi-acknowledgement. Using the same message format, we could pass repeated uint32 sequence numbers to mark those messages
    that we _have_ received. This would remove the worst-case wait time. The underlying assumption is that the connection is mostly-stable and mostly-reliable. That is, messages
    are more likely to be delivered than not delivered. With this assumption, we send less data marking the messages that are missing than we do marking the messages that are
    received. For a 1 in 100 failure rate on the connection, we would have to ack 99 messages in this multi acknowledgement message, or mark just 1 message as missing in the
    missing message. 
