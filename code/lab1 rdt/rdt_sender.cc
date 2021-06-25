/*
 * FILE: rdt_sender.cc
 * DESCRIPTION: Reliable data transfer sender.
 * NOTE: This implementation assumes there is no packet loss, corruption, or 
 *       reordering.  You will need to enhance it to deal with all these 
 *       situations.  In this implementation, the packet format is laid out as 
 *       the following:
 *       
 *       |<-  1 byte  ->|<-             the rest            ->|
 *       | payload size |<-             payload             ->|
 *
 *       The first byte of each packet indicates the size of the payload
 *       (excluding this single-byte header)
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <vector>

#include "rdt_struct.h"
#include "rdt_sender.h"
#include "protocol.h"


// #define __DEBUG__  
#ifdef __DEBUG__  
#define DEBUG(format, ...) printf(format, ##__VA_ARGS__) 
#else  
#define DEBUG(format,...)  
#endif  
static seq_sz next_seq;      /* used to calculate the seq of every frame, the number of the total frame is next_seq*/
static int nwindow; /* the number of frame in window*/
static std::vector<struct packet> buffer;

static void form_data_frame(struct packet *pkt, data_sz *data, size_sz size, seq_sz seq)
{
    data_frame *frm = (data_frame *)pkt;
    frm->size = size;
    frm->seq = seq;
    memcpy(frm->info, data, size);
    frm->checksum = cal_checksum(pkt);}

static void form_first_data_frame(struct packet *pkt, data_sz *data, size_sz size, size_sz total_size, seq_sz seq)
{
    data_first_frame *frm = (data_first_frame *)pkt;
    frm->size = size;
    frm->seq = seq;
    frm->total_size = total_size;
    memcpy(frm->info, data, size);
    frm->checksum = cal_checksum(pkt);
}

/* if buffer is not empty, and window is empty, send data */
static void send_data()
{      
    while (nwindow < WINDOW_SIZE && nwindow<(int)buffer.size())
    {
        nwindow++;
        struct packet pkt = buffer[nwindow-1];
        DEBUG("[Sender] [send frame] nwindow = %d, seq = %d \n",nwindow, ((data_frame*)&pkt)->seq);
        Sender_ToLowerLayer(&pkt);        
    }
}
/* sender initialization, called once at the very beginning */
void Sender_Init()
{
    fprintf(stdout, "At %.2fs: sender initializing ...\n", GetSimulationTime());
    next_seq = 0;
    nwindow = 0;
}

/* sender finalization, called once at the very end.
   you may find that you don't need it, in which case you can leave it blank.
   in certain cases, you might want to take this opportunity to release some 
   memory you allocated in Sender_init(). */
void Sender_Final()
{
    fprintf(stdout, "At %.2fs: sender finalizing ...\n", GetSimulationTime());
}

/* event handler, called when a message is passed from the upper layer at the 
   sender */
void Sender_FromUpperLayer(struct message *msg)
{
    struct packet pkt;

    int cursor = 0;

    /*put the first frame*/
    if(msg->size >(int)(MAX_PKT_SIZE - sizeof(size_sz)))
    {
        form_first_data_frame(&pkt,(data_sz*)msg->data,MAX_PKT_SIZE-sizeof(size_sz),msg->size,next_seq);
        next_seq++;
        buffer.push_back(pkt);
        cursor += MAX_PKT_SIZE-sizeof(size_sz);
    }
    else{
        form_first_data_frame(&pkt,(data_sz*)msg->data,msg->size,msg->size,next_seq);
        next_seq++;
        buffer.push_back(pkt);
        cursor += msg->size;
    }
    while (msg->size - cursor > MAX_PKT_SIZE)
    {
        /* fill in the buffer */
        form_data_frame(&pkt, (data_sz *)msg->data + cursor, MAX_PKT_SIZE, next_seq);

        next_seq ++; /*in case of overflowing*/
        buffer.push_back(pkt);
        /* move the cursor */
        cursor += MAX_PKT_SIZE;
    }

    /* send out the last packet */
    if (msg->size > cursor)
    {
        /* fill in the packet */
        form_data_frame(&pkt, (data_sz *)msg->data + cursor, msg->size-cursor, next_seq);
    
        next_seq ++; /*in case of overflowing*/
        buffer.push_back(pkt);
    }

    if (Sender_isTimerSet())
        return;

    /* if there is no timer -> window is empty*/
    Sender_StartTimer(TIMEOUT);
    send_data();
    if (nwindow == 0)
        Sender_StopTimer();
    
}

/* event handler, called when a packet is passed from the lower layer at the 
   sender */
void Sender_FromLowerLayer(struct packet *pkt)
{
    if(buffer.size()==0)
    {
        Sender_StopTimer();
        return;
    }
    control_frame *frm = (control_frame *)pkt;
    check_sz cksm = frm->checksum;
    if (cksm != cal_checksum(pkt))
        return;

    seq_sz ack = frm->ack;

    /* ack should be the seq number in window */
    seq_sz min_ack = ((data_frame *)&buffer.front())->seq;
    seq_sz max_ack = ((data_frame *)&(buffer[nwindow-1]))->seq;

    DEBUG("[Sender] [receiver ack] ack = %d, min_ack = %d, max_ack = %d\n",ack,min_ack,max_ack);

    if (ack>max_ack || ack<min_ack)
        return;

    while (min_ack <= ack)
    {
        nwindow--;
        DEBUG("[Sender] [pop window] min_ack = %d, ack = %d, seq of moved frame = %d,nwindow = %d,buffer_size = %ld\n",min_ack,ack,((data_frame*)&buffer.front())->seq,nwindow,buffer.size());
        buffer.erase(buffer.begin());
        min_ack ++;
    }

    Sender_StartTimer(TIMEOUT);
    send_data();
    if (nwindow == 0)
        Sender_StopTimer();
}

/* event handler, called when the timer expires */
/* resend the whole window*/
void Sender_Timeout()
{
    DEBUG("[Sender] [timeout] nwindow = %d, ",nwindow);
    if(nwindow==0)
        return;
    
    Sender_StartTimer(TIMEOUT);
    for (int i = 0;i<nwindow;i++)
    {
        DEBUG("%d ",((data_frame*)&buffer[i])->seq);
        Sender_ToLowerLayer(&buffer[i]);
    }
    DEBUG("\n");
}
