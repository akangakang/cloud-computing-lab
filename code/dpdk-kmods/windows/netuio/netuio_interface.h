/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

 /**
  * @file netuio kernel driver interface
  */

#ifndef NETUIO_INTERFACE_H
#define NETUIO_INTERFACE_H

/**
 * All structures in this file are packed on an 8B boundary.
 */
#pragma pack(push)
#pragma pack(8)

/**
 * Define an Interface Guid so that any app can find the device and talk to it.
 */
DEFINE_GUID (GUID_DEVINTERFACE_netUIO, 0x08336f60,0x0679,0x4c6c,0x85,0xd2,0xae,0x7c,0xed,0x65,0xff,0xf7); // {08336f60-0679-4c6c-85d2-ae7ced65fff7}

/**
 * Device name definitions
 */
#define NETUIO_DEVICE_SYMBOLIC_LINK_UNICODE    L"\\DosDevices\\netuio"
#define NETUIO_MAX_SYMLINK_LEN                 255

/**
 * IOCTL_NETUIO_MAP_HW_INTO_USERSPACE is used for mapping the device registers
 * into userspace. It returns the physical address, virtual address
 * and the size of the memory region where the BARs were mapped.
 */
#define IOCTL_NETUIO_MAP_HW_INTO_USERSPACE CTL_CODE(FILE_DEVICE_NETWORK, 51, METHOD_BUFFERED, FILE_READ_ACCESS | FILE_WRITE_ACCESS)

/**
 * IOCTL_NETUIO_PCI_CONFIG_IO is used to read/write from/into the device
 * configuration space.
 *
 * Input:
 *   - the operation type (read/write)
 *   - the offset into the device data where the operation begins
 *   - the length of data to read/write.
 *   - in case of a write operation, the data to be written to the device
 *     configuration space.
 *
 * Output:
 *   - in case of a read operation, the output buffer is filled
 *     with the data read from the configuration space.
 */
#define IOCTL_NETUIO_PCI_CONFIG_IO        CTL_CODE(FILE_DEVICE_NETWORK, 52, METHOD_BUFFERED, FILE_READ_ACCESS | FILE_WRITE_ACCESS)

struct mem_region {
    UINT64           size;       /**< memory region size */
    LARGE_INTEGER    phys_addr;  /**< physical address of the memory region */
    PVOID            virt_addr;  /**< virtual address of the memory region */
    PVOID            user_mapped_virt_addr;  /**< virtual address of the region mapped into user process context */
};

enum pci_io {
    PCI_IO_READ = 0,
    PCI_IO_WRITE = 1
};

#define PCI_MAX_BAR 6

struct device_info
{
    struct mem_region   hw[PCI_MAX_BAR];
};

struct dpdk_pci_config_io
{
    UINT32              offset; /**< offset into the device config space where the reading/writing starts */
    UINT8               op; /**< operation type: read or write */
    UINT32              access_size; /**< 1, 2, 4, or 8 bytes */

    union dpdk_pci_config_io_data {
        UINT8			u8;
        UINT16			u16;
        UINT32			u32;
        UINT64			u64;
    } data; /**< Data to be written, in case of write operations */
};

#pragma pack(pop)

#endif // NETUIO_INTERFACE_H
