## Instructions to run

#### Success case:

1. Run the server
   ```
   go run ./cmd/server/ -port 8080
   ```

2. In a different session/terminal window run the client
   ```
   go run ./cmd/client/ -port 8080 -n 10
   ```

3. You should see 10 numbers generated and passed from Server to Client. The checksum should also match.
   Example:
   ```
    $ go run ./cmd/client/ -port 8080 -n 10
    2023/08/03 13:29:51 [0] number: 1051999831
    2023/08/03 13:29:52 [1] number: 3899366106
    2023/08/03 13:29:53 [2] number: 3514390649
    2023/08/03 13:29:54 [3] number: 2776206725
    2023/08/03 13:29:55 [4] number: 664399516
    2023/08/03 13:29:56 [5] number: 53824273
    2023/08/03 13:29:57 [6] number: 268394589
    2023/08/03 13:29:58 [7] number: 2032633658
    2023/08/03 13:29:59 [8] number: 2840642574
    2023/08/03 13:30:00 [9] number: 78856040
    2023/08/03 13:30:01 [10] checksum: 8d52dbc1b279ea97fb6a5854f0b5512db1280357
    2023/08/03 13:30:01 [OK] checksum match
    2023/08/03 13:30:01 8d52dbc1b279ea97fb6a5854f0b5512db1280357 = 8d52dbc1b279ea97fb6a5854f0b5512db1280357
   ```

#### Resume:

1. Run the server as before

2. Run the client with a static id, but Ctrl-C the client half way through the number generation.
   ```
    $ go run ./cmd/client/ -port 8080 -n 10 -id=my-client-id-abc
   2023/08/03 13:26:59 [0] number: 2089351891
   2023/08/03 13:27:00 [1] number: 1856068399
   2023/08/03 13:27:01 [2] number: 976166314
   2023/08/03 13:27:02 [3] number: 3012547545
   ^C
   2023/08/03 13:27:02 exiting
   ```

3. Resume the client with the same id.
   ```
    $ go run ./cmd/client/ -port 8080 -n 10 -id=my-client-id-abc
    2023/08/03 13:27:04 resumed data from store, series: [2089351891 1856068399 976166314 3012547545 0 0 0 0 0 0]
    2023/08/03 13:27:04 [5] number: 4109345573
    2023/08/03 13:27:05 [6] number: 3038305399
    2023/08/03 13:27:06 [7] number: 2740718436
    2023/08/03 13:27:07 [8] number: 330912612
    2023/08/03 13:27:08 [9] number: 2276818533
    2023/08/03 13:27:09 [4] number: 1899243751
    2023/08/03 13:27:09 [10] checksum: 08bce3a368f6a8e706c05df4dbb8e0644bf658b5
    2023/08/03 13:27:09 [OK] checksum match
    2023/08/03 13:27:09 08bce3a368f6a8e706c05df4dbb8e0644bf658b5 = 08bce3a368f6a8e706c05df4dbb8e0644bf658b5
   ```
   Depending on the timing of the Ctrl-C, you will likely also see the messages appearing in the wrong order. See message `[4]` in the above example.

   The client flushes the data to a file on exit. The file is `/tmp/{client-id}.state`. You can artificially force the checksum mismatch by editing this file before resuming the client. The server only keeps its state in memory.

## Message format

This protocol is [TLV (type-length-value)](https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value) encoded data stream in binary format.

All numbers are BigEndian encoded.

`Connect`

    `Byte3('CON')`
        Identifies the message as a new connection

    `Int32`
        Number of numbers to receive

    `Int32`
        Length of the message content in bytes, not including self

    `String`
        Client ID

- Connect can be used to both initialise a new connection, and to resume a dropped connection.
- The connect message contains the `n` number of numbers to receive in sequence, and the client id.
- If the client id is unknown to the server, it's a new connection, if the client id is known to the server then the sequence should be resumed from the last acknowledged message.

`Number`

    `Byte3('NUM')`
        Identifies the message as a number

    `Int32`
        Sequence number of this number in the series

    `Int32`
       The payload number.

- Number is a single number in the sequence. The Number message contains both the number field, and a sequence number.
- The sequence number is incrementing for each number in the sequence. First message 1, second message 2, etc.
- The client should expect it could receive the same number in the sequence multiple times.

`Acknowledge`

    `Byte3('ACK')`
        Identifies the message as a sequence acknowledgement

    `Int32`
        Sequence number to acknowledge

- Acknowledge message acknowledges an element in the series of numbers.
- The server **must not** wait for an acknowledgement on the previous message before sending the next message.
- The client should acknowledge the numbers in the sequence.
- The server should expect it could receive an acknowledgement for the same sequence number multiple times.
- The server assumes that messages are undelivered until it receives an Acknowledge for that messages sequence number.
- The server should continue with the sequence of numbers. Once the sequence has completed, the server should use the remaining ticks to send the unacknowledged messages.

`Checksum`

    `Byte3('CHK')`
        Identifies the message as a Checksum

    `Int32`
        Sequence number of this number in the series

    `Int32`
        Length of the checksum data in bytes, not including self

    `String`
        Checksum data

- Checksum message indicates the end of the sequence, and carries a checksum value.
- The checksum is [....]
- The client must acknowledge the checksum message. An acknowledgement of the checksum message indicates to the server that its job is done.
- If the server exhausts its series of numbers, and is missing acknowledgements for some numbers in the series, it should use its next server-ticks to resend those numbers.
- If the server has received acknowledgements for all the numbers in the series, but is missing the acknowledgement for the checksum, the server should resend the checksum until the
  acknowledgement is received.
- The client should assume that message can be delivered out of order. With this in mind, when receiving the checksum message (which marks the end of the stream), the client should
  keep track of all the missing sequence numbers up to the sequence number of the checksum message. Receiving the checksum message marks the last most sequence number in the stream
  of messages.

## Open questions

- One of the requirements reads:

  _"The timing of the messages should be under control of the server (that is: information may be sent by the client as part of initiating a
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

  The `Checksum` message marks the end of the series.
  - If the client has received a checksum message then it knows the end of the sequence numbers and can find the missing sequence
  numbers.

  - If the client has not received a checksum message, then it can find the missing sequence numbers that it already knows about AND it can include the highest sequence
  number +1.

  This is safe because the client knows that it must receive at least one more message, up until the checksum message is received. The draw back here is that for `S`
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

## Server

The server has a list of message that it needs to send, it will work its way though that list from start to finish.
In the background, on the same connection, the server receives acknowledge messages. It uses these acknowledgements to mark which messages have been successfully received by the
client. This is the 'initial send' phase.

Once the server has completed the first send on all the messages in the series, it revisits the series in order, re-sending each message that is not marked as acknowledged.
The server will repeat this process until all the messages in the series are acknowledged. This is the 'resend' phase.

When a client re-connects to the server, the server enters the 'resend' phase, working its way through all unacknowledged messages and sending them, continuing until all messages
in the series are acknowledged.

## Client

The client connects to the server passing a client-id and `n` numbers to receive in a series.
The client should acknowledge the message it receives, it can send an acknowledgement at any time.

When the client has received all the numbers in the series, and a checksum, compute the checksum on the messages received and exit, printing the status.

## Tests

The only automated / unit tests that are included in this project (so far) are for the message encoding / decoding. The client and server share the same message structure
definitions, so I've tested that encoding a message to bytes, and decoding it back into a struct works as expected. This ensures the client and server will understand each others'
messages. 
