# Lab 1 - Reliable Data Transport Protocol

姓名：刘书畅

学号：518021910789

邮箱：liushuchang0609@sjtu.edu.cn



**Note**: 由于新增文件`protocol.h`,所以修改了`makefile`



## 策略设计

采用**Go-Back-N**策略

### 1. Frame设计

根据需要设计了三种`frame`

1. `data_first_frame`:`Sender`传给`Receiver`的第一个数据包

   ```c
   typedef struct
   {
       size_sz size;
       seq_sz seq;
       size_sz total_size;
       data_sz info[MAX_PKT_SIZE - sizeof(size_sz)];
       check_sz checksum;
   } data_first_frame;
   ```

   ```
   +--------+-------+------------+------------------+----------+
   |        |       |            |                  |          |
   |  size  |  seq  | total_size |       info       | checksum |
   |        |       |            |                  |          |
   +--------+-------+------------+------------------+----------+
   
     4 byte   4 byte    4 byte         114 byte        2 byte
   
   ```

   `size `:用于存储`info`中有效数据的大小

   `seq`:用于存储该`frame`的序列号

   `total_size`：用于存储整个`message`的大小

   `checksum`：校验码
   
2. `data_frame`:`Sender`传给`Receiver`的数据包(非第一个)

   ```c
   typedef struct
   {
       size_sz size;
       seq_sz seq;
       data_sz info[MAX_PKT_SIZE];
       check_sz checksum;
   } data_frame;
   ```

   ```
   +--------+-------+-------------------+------------+
   |        |       |                   |            |
   |  size  |  seq  |       info        |  checksum  |
   |        |       |                   |            |
   +--------+-------+-------------------+------------+
   
     4 byte   4 byte      118 byte          2 byte
   
   ```

   `size `:用于存储`info`中有效数据的大小

   `seq`:用于存储该`frame`的序列号

   `checksum`：校验码
   
3. `control_frame`:`Receiver`传给`Sender`的控制包

   ```c
   typedef struct
   {
       seq_sz ack;
       data_sz meaningless[RDT_PKTSIZE - C_HEAD_SIZE - TAIL_SIZE];
       check_sz checksum;
   } control_frame;
   ```

   ```
   +---------+---------------------+--------------+
   |         |                     |              |
   |   ack   |     meaningless     |   checksum   |
   |         |                     |              |
   +---------+---------------------+--------------+
   
      4 byte        122 byte            2 byte
   
   ```

   `ack`:用于存储ack

   `meaningless`：无实际意义

   `checksum`：校验码   



### 2. Sender设计

#### 1.设计

采用**Go-Back-N**策略

##### `Sender_FromUpperLayer`

当数据链路层从网络层收到数据时，为了提升`Sender_FromUpperLayer`函数的速度，采用`buffer`设计

先将`message`中的数据都封装成`frame`存在`buffer`里，如果`window`中有空位，则按顺序发送`buffer`中的`frame`，否则`Sender_FromUpperLayer`直接返回

注意：第一个数据包要特殊处理，将`message`的大小存入`data_first_frame`中的`total_size`



##### `Sender_FromLowerLayer`

数据链路层从物理层收到序号为`ack`的控制包，说明`seq<=ack`的数据包已被正常收到，可以从`window`中移出，可以进行新一轮的发包，将`window`填满



##### `Sender_Timeout`

一个`window`的包共用一个计时器，如果里面有一个包出错超时，则整个`window`超时，一个`window`的`frame`都要重发



#### 2.实现

使用数据结构`vector`存储`buffer`

为了减少数据拷贝，不使用新的结构存储`window`，而是用`nwindow`记录`window`中`frame`的数。即`buffer`中的前`nwindow`个包是`window`中的，是已经发送给`Receiver`的包

当收到来自`Receiver`的控制包，确认已经收到时，可以将包从`buffer`中弹出，`nwindow`数量相应减少



### 3. Receiver设计

#### 1. 设计

`Receiver`接收数据包，组合成`message`，完整时向上发送给网络层

使用`window`存储乱序到达的数据包，最多存储`WINDOW_SIZE`（实现中设置为10）个数据包。若到达的数据包是需要的数据包(`seq == current_message`)，则存入`current_message`，否则存入`window`

每次收到数据包后检查`window`中是否有`seq`为`seq_expected`的数据包，有的话则移动`window`，将数据存入`current_message`，并发送`ack`为`seq_expected`的控制包给`Sender`

#### 2. 实现

使用`cursor`记录`current_message`中所需包的存储位置。当`cursor == current_message->size`时说明`current_message`已经完整，可以将数据向上发送给网络层

使用数据结构`vector`存储`window`，`window_valid`，大小为`WINDOW_SIZE`。`window_valid`记录是否存储了合法数据包

用`seq_expected`记录所需的数据包的序列号。

收到序号为`seq`的数据包：

1. `seq == current_message`

   则直接将数据包的`info`存入`current_message`的`cursor`位置,`seq_expected`增加

2. `seq_expected < seq_received && seq_received < seq_expected + WINDOW_SIZE)`

   将数据包存入`window[seq_received - seq_expected - 1]`处，并将`window_valid`相应位置设为1，`seq_expected`增加

之后移动`window`，若`window`中存储了`seq_expected`的数据包，则存入`current_message`,继续移动

最后向`Sender`发送`ack`为`seq_expected-1`的控制包

### 4. Checksum设计

采用16 位 Internet checksum

由于数据包和控制包的数据都将`checksum`存在最后两个字节的位置，所以计算`packet`的`0`-`RDT_PKTSIZE-TAIL_SIZE`数据获得`sum`再取反，得到16位`checksum`



## 运行效果

```
os@ubuntu:~/Desktop/cloud-computing-lab/rdt$ ./rdt_sim 1000 0.1 100 0.15 0.15 0.15 0
## Reliable data transfer simulation with:
        simulation time is 1000.000 seconds
        average message arrival interval is 0.100 seconds
        average message size is 100 bytes
        average out-of-order delivery rate is 15.00%
        average loss rate is 15.00%
        average corrupt rate is 15.00%
        tracing level is 0
Please review these inputs and press <enter> to proceed.

At 0.00s: sender initializing ...
At 0.00s: receiver initializing ...
At 1022.35s: sender finalizing ...
At 1022.35s: receiver finalizing ...

## Simulation completed at time 1022.35s with
        998090 characters sent
        998090 characters delivered
        50119 packets passed between the sender and the receiver
## Congratulations! This session is error-free, loss-free, and in order.
```

```
os@ubuntu:~/Desktop/cloud-computing-lab/rdt$ ./rdt_sim 1000 0.1 100 0.3 0.3 0.3 0
## Reliable data transfer simulation with:
        simulation time is 1000.000 seconds
        average message arrival interval is 0.100 seconds
        average message size is 100 bytes
        average out-of-order delivery rate is 30.00%
        average loss rate is 30.00%
        average corrupt rate is 30.00%
        tracing level is 0
Please review these inputs and press <enter> to proceed.

At 0.00s: sender initializing ...
At 0.00s: receiver initializing ...
At 1848.38s: sender finalizing ...
At 1848.38s: receiver finalizing ...

## Simulation completed at time 1848.38s with
        1006321 characters sent
        1006321 characters delivered
        62237 packets passed between the sender and the receiver
## Congratulations! This session is error-free, loss-free, and in order.
```

