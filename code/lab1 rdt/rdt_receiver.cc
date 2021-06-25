/*
 * FILE: rdt_receiver.cc
 * DESCRIPTION: Reliable data transfer receiver.
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
#include "rdt_receiver.h"
#include "protocol.h"

// #define __DEBUG__  
#ifdef __DEBUG__  
// #define DEBUG(format,...) printf("File: "__FILE__", Line: %05d: "format"/n", __LINE__, ##__VA_ARGS__) 
#define DEBUG(format, ...) printf(format, ##__VA_ARGS__) 
#else  
#define DEBUG(format,...)  
#endif 

static message *current_message;
static int cursor;
static std::vector<struct packet> window(10);
static std::vector<int> window_valid(WINDOW_SIZE, 0);
static seq_sz seq_expected;

static void Receiver_form_and_send_ack(seq_sz seq)
{
    struct packet pkt;
    control_frame *c_frm = (control_frame *)&pkt;

    DEBUG("[Receiver] [send ack] seq_expected = %d, ack = %d\n", seq_expected, seq);

    c_frm->ack = seq;
    check_sz cksm = cal_checksum(&pkt);
    c_frm->checksum = cksm;
    Receiver_ToLowerLayer(&pkt);
}

static void move_window()
{
    // DEBUG("[Receiver] [move window]\n");
    if (cursor == current_message->size)
    {
        DEBUG("[Receiver] [finish msg]\n");
        Receiver_ToUpperLayer(current_message);
        cursor = 0;
    }

    while (window_valid.front() == 1)
    {
        if (cursor == 0)
        {
            if (current_message->size != 0)
            {
                current_message->size = 0;
                free(current_message->data);
            }

            data_first_frame *d_f_frm = (data_first_frame *)&window.front();

            DEBUG("[Receiver] [move window] [first packet] seq_expected = %d, seq = %d, total_size = %d ,size = %d \n", seq_expected, d_f_frm->seq, d_f_frm->total_size, d_f_frm->size);

            current_message->size = d_f_frm->total_size;
            current_message->data = (char *)malloc(current_message->size);

            memcpy(current_message->data + cursor, d_f_frm->info, d_f_frm->size);
            cursor += d_f_frm->size;
        }
        else
        {
            data_frame *d_frm = (data_frame *)&window.front();

            DEBUG("[Receiver] [move window] [not first packet] seq_expected = %d, seq = %d, size = %d \n", seq_expected, d_frm->seq, d_frm->size);

            memcpy(current_message->data + cursor, d_frm->info, d_frm->size);
            cursor += d_frm->size;
        }

        window.erase(window.begin());
        struct packet pkt;
        window.push_back(pkt);
        window_valid.erase(window_valid.begin());
        window_valid.push_back(0);
        seq_expected++;

        if (cursor == current_message->size)
        {
            DEBUG("[Receiver] [finish msg]\n");
            Receiver_ToUpperLayer(current_message);
            cursor = 0;
        }
    }

    Receiver_form_and_send_ack(seq_expected - 1);
}

/* receiver initialization, called once at the very beginning */
void Receiver_Init()
{
    fprintf(stdout, "At %.2fs: receiver initializing ...\n", GetSimulationTime());

    current_message = (message *)malloc(sizeof(message));
    memset(current_message, 0, sizeof(message));

    cursor = 0;
    seq_expected = 0;
}

/* receiver finalization, called once at the very end.
   you may find that you don't need it, in which case you can leave it blank.
   in certain cases, you might want to use this opportunity to release some 
   memory you allocated in Receiver_init(). */
void Receiver_Final()
{
    fprintf(stdout, "At %.2fs: receiver finalizing ...\n", GetSimulationTime());
    free(current_message);
}

/* event handler, called when a packet is passed from the lower layer at the 
   receiver */
void Receiver_FromLowerLayer(struct packet *pkt)
{
    data_frame *frm = (data_frame *)pkt;
    check_sz cksm = frm->checksum;

    if (cksm != cal_checksum(pkt))
        return;

    seq_sz seq_received = frm->seq;
    if (seq_expected < seq_received && seq_received < seq_expected + WINDOW_SIZE)
    {
        DEBUG("[Receiver] [put in buffer] seq_expected = %d, seq = %d \n", seq_expected, seq_received);
        memcpy(window[seq_received - seq_expected - 1].data, pkt->data, RDT_PKTSIZE);
        window_valid[seq_received - seq_expected - 1] = 1;

        if(seq_expected!=0)
            Receiver_form_and_send_ack(seq_expected - 1);
        return;
    }
    else if (seq_expected == seq_received)
    {
        if (cursor == 0)
        {
            if (current_message->size != 0)
            {
                current_message->size = 0;
                free(current_message->data);
            }

            data_first_frame *d_f_frm = (data_first_frame *)pkt;

            DEBUG("[Receiver] [first packet] seq_expected = %d, seq = %d, total_size = %d ,size = %d \n", seq_expected, seq_received, d_f_frm->total_size, d_f_frm->size);

            current_message->size = d_f_frm->total_size;
            current_message->data = (char *)malloc(current_message->size);

            memcpy(current_message->data + cursor, d_f_frm->info, d_f_frm->size);
            cursor += d_f_frm->size;
        }
        else
        {
            data_frame *d_frm = (data_frame *)pkt;

            DEBUG("[Receiver] [not first packet] seq_expected = %d, seq = %d, size = %d \n", seq_expected, seq_received, d_frm->size);

            memcpy(current_message->data + cursor, d_frm->info, d_frm->size);
            cursor += d_frm->size;
        }
        seq_expected++;
        

        move_window();
        if (window_valid.front() != 1)
        {
            window.erase(window.begin());
            struct packet pkt;
            window.push_back(pkt);
            window_valid.erase(window_valid.begin());
            window_valid.push_back(0);
        }
    }
    else
    {
        Receiver_form_and_send_ack(seq_expected - 1);
    }
}