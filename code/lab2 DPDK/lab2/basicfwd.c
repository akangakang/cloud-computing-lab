/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2010-2015 Intel Corporation
 */

#include <stdint.h>
#include <inttypes.h>
#include <rte_eal.h>
#include <rte_ethdev.h>
#include <rte_cycles.h>
#include <rte_lcore.h>
#include <rte_mbuf.h>

#define RX_RING_SIZE 1024
#define TX_RING_SIZE 1024

#define NUM_MBUFS 8191
#define MBUF_CACHE_SIZE 250
#define BURST_SIZE 32

static const struct rte_eth_conf port_conf_default = {
	.rxmode = {
		.max_rx_pkt_len = RTE_ETHER_MAX_LEN,
	},
};

/* basicfwd.c: Basic DPDK skeleton forwarding example. */

/*
 * Initializes a given port using global settings and with the RX buffers
 * coming from the mbuf_pool passed as a parameter.
 */
static inline int
port_init(uint16_t port, struct rte_mempool *mbuf_pool)
{
	struct rte_eth_conf port_conf = port_conf_default;
	const uint16_t rx_rings = 1, tx_rings = 1;
	uint16_t nb_rxd = RX_RING_SIZE;
	uint16_t nb_txd = TX_RING_SIZE;
	int retval;
	uint16_t q;
	struct rte_eth_dev_info dev_info;
	struct rte_eth_txconf txconf;

	if (!rte_eth_dev_is_valid_port(port))
		return -1;

	retval = rte_eth_dev_info_get(port, &dev_info);
	if (retval != 0)
	{
		printf("Error during getting device (port %u) info: %s\n",
			   port, strerror(-retval));
		return retval;
	}

	if (dev_info.tx_offload_capa & DEV_TX_OFFLOAD_MBUF_FAST_FREE)
		port_conf.txmode.offloads |=
			DEV_TX_OFFLOAD_MBUF_FAST_FREE;

	/* Configure the Ethernet device. */
	retval = rte_eth_dev_configure(port, rx_rings, tx_rings, &port_conf);
	if (retval != 0)
		return retval;

	retval = rte_eth_dev_adjust_nb_rx_tx_desc(port, &nb_rxd, &nb_txd);
	if (retval != 0)
		return retval;

	/* Allocate and set up 1 RX queue per Ethernet port. */
	for (q = 0; q < rx_rings; q++)
	{
		retval = rte_eth_rx_queue_setup(port, q, nb_rxd,
										rte_eth_dev_socket_id(port), NULL, mbuf_pool);
		if (retval < 0)
			return retval;
	}

	txconf = dev_info.default_txconf;
	txconf.offloads = port_conf.txmode.offloads;
	/* Allocate and set up 1 TX queue per Ethernet port. */
	for (q = 0; q < tx_rings; q++)
	{
		retval = rte_eth_tx_queue_setup(port, q, nb_txd,
										rte_eth_dev_socket_id(port), &txconf);
		if (retval < 0)
			return retval;
	}

	/* Start the Ethernet port. */
	retval = rte_eth_dev_start(port);
	if (retval < 0)
		return retval;

	/* Display the port MAC address. */
	struct rte_ether_addr addr;
	retval = rte_eth_macaddr_get(port, &addr);
	if (retval != 0)
		return retval;

	printf("Port %u MAC: %02" PRIx8 " %02" PRIx8 " %02" PRIx8
		   " %02" PRIx8 " %02" PRIx8 " %02" PRIx8 "\n",
		   port,
		   addr.addr_bytes[0], addr.addr_bytes[1],
		   addr.addr_bytes[2], addr.addr_bytes[3],
		   addr.addr_bytes[4], addr.addr_bytes[5]);

	/* Enable RX in promiscuous mode for the Ethernet device. */
	retval = rte_eth_promiscuous_enable(port);
	if (retval != 0)
		return retval;

	return 0;
}

static void form_ether_hdr(struct rte_ether_hdr *eth_hdr, struct rte_ether_addr s_addr, struct rte_ether_addr d_addr, uint16_t e_type)
{
	eth_hdr->d_addr = d_addr;
	eth_hdr->s_addr = s_addr;
	eth_hdr->ether_type = rte_cpu_to_be_16(e_type);
}

static void form_ip_hdr(struct rte_ipv4_hdr *ip_hdr, uint8_t version_ihl, uint8_t type_of_service, rte_be16_t total_length, rte_be16_t packet_id, rte_be16_t fragment_offset, uint8_t time_to_live, uint8_t next_proto_id, rte_be16_t hdr_checksum, rte_be32_t src_addr, rte_be32_t dst_addr)
{
	ip_hdr->version_ihl = version_ihl;
	ip_hdr->type_of_service = type_of_service;
	ip_hdr->total_length = rte_cpu_to_be_16(total_length);
	ip_hdr->packet_id = rte_cpu_to_be_16(packet_id);
	ip_hdr->fragment_offset = rte_cpu_to_be_16(fragment_offset);
	ip_hdr->time_to_live = time_to_live;
	ip_hdr->next_proto_id = next_proto_id;
	ip_hdr->src_addr = rte_cpu_to_be_32(src_addr);
	ip_hdr->dst_addr = rte_cpu_to_be_32(dst_addr);
	ip_hdr->hdr_checksum = rte_cpu_to_be_16(hdr_checksum);
}

static void form_udp_hdr(struct rte_udp_hdr *udp_hdr, rte_be16_t src_port, rte_be16_t dst_port, rte_be16_t dgram_len, rte_be16_t dgram_cksum)
{
	udp_hdr->src_port = rte_cpu_to_be_16(src_port);
	udp_hdr->dst_port = rte_cpu_to_be_16(dst_port);
	udp_hdr->dgram_len = rte_cpu_to_be_16(dgram_len);
	udp_hdr->dgram_cksum = rte_cpu_to_be_16(dgram_cksum);
}
/*
 * The main function, which does initialization and calls the per-lcore
 * functions.
 */
int main(int argc, char *argv[])
{
	struct rte_mempool *mbuf_pool;
	// unsigned nb_ports;
	uint16_t portid;

	/* Initialize the Environment Abstraction Layer (EAL). */
	int ret = rte_eal_init(argc, argv);
	if (ret < 0)
		rte_exit(EXIT_FAILURE, "Error with EAL initialization\n");

	argc -= ret;
	argv += ret;

	/* Creates a new mempool in memory to hold the mbufs. */
	mbuf_pool = rte_pktmbuf_pool_create("MBUF_POOL", NUM_MBUFS,
										MBUF_CACHE_SIZE, 0, RTE_MBUF_DEFAULT_BUF_SIZE, rte_socket_id());

	if (mbuf_pool == NULL)
		rte_exit(EXIT_FAILURE, "Cannot create mbuf pool\n");

	/* Initialize all ports. */
	RTE_ETH_FOREACH_DEV(portid)
	if (port_init(portid, mbuf_pool) != 0)
		rte_exit(EXIT_FAILURE, "Cannot init port %" PRIu16 "\n",
				 portid);

	struct rte_mbuf *bufs[BURST_SIZE];
	int i;
	for (i = 0; i < BURST_SIZE; i++)
	{
		bufs[i] = rte_pktmbuf_alloc(mbuf_pool);
		struct rte_ether_hdr *eth_hdr = rte_pktmbuf_mtod(bufs[i], struct rte_ether_hdr *);
		struct rte_ipv4_hdr *ip_hdr = (struct rte_ipv4_hdr *)(rte_pktmbuf_mtod(bufs[i], char *) + sizeof(struct rte_ether_hdr));
		struct rte_udp_hdr *udp_hdr = (struct rte_udp_hdr *)(rte_pktmbuf_mtod(bufs[i], char *) + sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr));

		char *data = (char *)(rte_pktmbuf_mtod(bufs[i], char *) + sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr));
		strcpy(data, "Hello! Finish Lab2!");

		struct rte_ether_addr s_addr = {{0x00, 0x0c, 0x29, 0xa0, 0x50, 0x45}};
		struct rte_ether_addr d_addr = {{0x00, 0x50, 0x56, 0xc0, 0x00, 0x01}};

		form_ether_hdr(eth_hdr, s_addr, d_addr, RTE_ETHER_TYPE_IPV4);
		form_ip_hdr(ip_hdr, RTE_IPV4_VHL_DEF, RTE_IPV4_HDR_DSCP_MASK, 28 + strlen(data), 0, 0, 64, 17, rte_ipv4_cksum(ip_hdr), RTE_IPV4(192, 168, 136, 9), RTE_IPV4(192, 168, 136, 1));
		form_udp_hdr(udp_hdr, 80, 8080, 8 + strlen(data), 1);

		bufs[i]->data_len = sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr) + strlen(data);
		bufs[i]->pkt_len = sizeof(struct rte_ether_hdr) + sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr) + strlen(data);
	}

	uint16_t nb_tx = rte_eth_tx_burst(0, 0, bufs, BURST_SIZE);
	printf("发送成功%d个包\n", nb_tx);

	/* Free any unsent packets. */
	if (unlikely(nb_tx < BURST_SIZE))
	{
		uint16_t buf;
		for (buf = nb_tx; buf < BURST_SIZE; buf++)
			rte_pktmbuf_free(bufs[buf]);
	}
}
