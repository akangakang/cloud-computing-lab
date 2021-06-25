/*
 * FILE: protocol.h
 * DESCRIPTION: The protocol.
 */

#ifndef _PROTOCOL_H_
#define _PROTOCOL_H_

#include "rdt_struct.h"
#include <string.h>

typedef unsigned short check_sz;
typedef unsigned int seq_sz;
typedef unsigned int size_sz;
typedef unsigned char data_sz;

#define D_HEAD_SIZE 8 //(sizeof(size_sz) + sizeof(seq_sz)) /* byte */
#define TAIL_SIZE 2 //(sizeof(check_sz))
#define C_HEAD_SIZE 4 //(sizeof(seq_sz))
#define MAX_PKT_SIZE (RDT_PKTSIZE - D_HEAD_SIZE - TAIL_SIZE)
#define TIMEOUT 0.3
#define WINDOW_SIZE 10


enum Bool
{
    False,
    True
};

typedef struct
{
    size_sz size;
    seq_sz seq;
    data_sz info[MAX_PKT_SIZE];
    check_sz checksum;
} data_frame;

typedef struct
{
    size_sz size;
    seq_sz seq;
    size_sz total_size;
    data_sz info[MAX_PKT_SIZE - sizeof(size_sz)];
    check_sz checksum;
} data_first_frame;


typedef struct
{
    seq_sz ack;
    data_sz meaningless[RDT_PKTSIZE - C_HEAD_SIZE - TAIL_SIZE];
    check_sz checksum;
} control_frame;

static check_sz cal_checksum(struct packet *pkt)
{
    unsigned long checksum = 0; 
    
    for (int i = 0; i < RDT_PKTSIZE-TAIL_SIZE; i += 2) {
        checksum += *(short *)(&(pkt->data[i]));
    }
    while (checksum >> 16) {
        checksum = (checksum >> 16) + (checksum & 0xffff);
    }
    return ~checksum;
}

#endif /* _PROTOCOL_H_ */