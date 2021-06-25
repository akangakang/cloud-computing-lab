#include "rte_common.h"
#include "rte_mbuf.h"
#include "rte_meter.h"
#include "rte_red.h"

#include "qos.h"

struct rte_meter_srtcm app_flows[APP_FLOWS_MAX];
struct rte_meter_srtcm_profile app_srtcm_profile[APP_FLOWS_MAX];
// struct rte_meter_srtcm_params app_srtcm_params[] = {
//     {.cir = 160000000, .cbs = 1600000, .ebs = 1000},
//     {.cir = 160000000, .cbs = 80000, .ebs = 500},
//     {.cir = 160000000, .cbs = 40000, .ebs =250},
//     {.cir = 160000000, .cbs = 20000, .ebs = 125},
// };
// struct rte_meter_srtcm_params app_srtcm_params[] = {
//     {.cir = 160000000, .cbs = 160000, .ebs = 0},
//     {.cir = 80000000, .cbs = 160000, .ebs =0},
//     {.cir = 40000000, .cbs = 160000, .ebs =0},
//     {.cir = 20000000, .cbs = 160000, .ebs = 0},
// };
struct rte_meter_srtcm_params app_srtcm_params[] = {
    {.cir = 160000000, .cbs = 160000, .ebs = 160000},
    {.cir = 80000000, .cbs = 80000, .ebs = 80000},
    {.cir = 40000000, .cbs = 40000, .ebs =40000},
    {.cir = 20000000, .cbs = 20000, .ebs = 20000},
};
struct rte_red_config app_red_params[3];
struct rte_red app_queue[APP_FLOWS_MAX];
uint64_t app_queue_size[APP_FLOWS_MAX] = {0};
uint64_t pretime = 0;

uint64_t cycle;
uint64_t hz;

/**
 * srTCM
 */
int qos_meter_init(void)
{
    cycle = rte_get_tsc_cycles();
    hz = rte_get_tsc_hz();

    int ret;
    for (int i = 0; i < APP_FLOWS_MAX; ++i)
    {
        ret = rte_meter_srtcm_profile_config(&app_srtcm_profile[i],
                                             &app_srtcm_params[i]);
        if (ret)
            return ret;
        ret = rte_meter_srtcm_config(&app_flows[i], &app_srtcm_profile[i]);
        if (ret)
        {
            return ret;
        }
    }
    return 0;
}

enum qos_color
qos_meter_run(uint32_t flow_id, uint32_t pkt_len, uint64_t time)
{
    time = time * hz / 1000000000 + cycle;
    return rte_meter_srtcm_color_blind_check(&app_flows[flow_id], &app_srtcm_profile[flow_id], time, pkt_len);
}

/**
 * WRED
 */

int qos_dropper_init(void)
{
    if (rte_red_config_init(&app_red_params[GREEN], RTE_RED_WQ_LOG2_MAX, 1022, 1023, 10))
        rte_panic("Cannot init GREEN config\n");
    if (rte_red_config_init(&app_red_params[YELLOW], RTE_RED_WQ_LOG2_MAX, 1022, 1023, 10))
        rte_panic("Cannot init YELLOW config\n");
    if (rte_red_config_init(&app_red_params[RED], RTE_RED_WQ_LOG2_MAX, 0, 1, 10))
        rte_panic("Cannot init RED config\n");

    int ret;
    for (int i = 0; i < APP_FLOWS_MAX; ++i)
    {
        ret = rte_red_rt_data_init(&app_queue[i]);
        /* Stop if error occurs */
        if (ret != 0)
        {
            return ret;
        }
    }
    return 0;
}

int qos_dropper_run(uint32_t flow_id, enum qos_color color, uint64_t time)
{
    if (time != app_queue[flow_id].q_time)
    {
        rte_red_mark_queue_empty(&app_queue[flow_id], time);
        app_queue_size[flow_id] = 0;
    }

    /* Make enqueue operation */
    int ret = rte_red_enqueue(&app_red_params[color], &app_queue[flow_id], app_queue_size[flow_id], time);
    if (ret)
    {
        return 1;
    }
    else
    {
        ++app_queue_size[flow_id];
        return 0;
    }
}