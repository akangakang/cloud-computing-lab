/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

#ifndef NETUIO_DEV_H
#define NETUIO_DEV_H

#include "netuio_interface.h"

struct pci_bar {
    PHYSICAL_ADDRESS base_addr;
    PVOID            virt_addr;
    UINT64           size;
};

struct mem_map_region {
    PMDL               mdl;    /**< MDL describing the memory region */
    struct mem_region  mem;    /**< Memory region details */
};

struct dev_addr {
    ULONG   bus_num;
    USHORT  dev_num;
    USHORT  func_num;
};

/**
 * The device context performs the same job as a WDM device extension in the driver frameworks
 */
typedef struct _NETUIO_CONTEXT_DATA
{
    WDFDEVICE               wdf_device;             /**< WDF device handle to the FDO */
    BUS_INTERFACE_STANDARD  bus_interface;          /**< Bus interface for config space access */
    struct pci_bar          bar[PCI_MAX_BAR];       /**< device BARs */
    struct dev_addr         addr;                   /**< B:D:F details of device */
    struct mem_map_region   dpdk_hw[PCI_MAX_BAR];   /**< mapped region for the device's register space */
} NETUIO_CONTEXT_DATA, *PNETUIO_CONTEXT_DATA;


/**
 * This macro will generate an inline function called DeviceGetContext
 * which will be used to get a pointer to the device context memory in a
 * type safe manner.
 */
WDF_DECLARE_CONTEXT_TYPE_WITH_NAME(NETUIO_CONTEXT_DATA, netuio_get_context_data)

typedef struct
{
    BOOLEAN bMapped;     /**< value is set to TRUE if the User-mode mapping was done for this file object. */
}  NETUIO_FILE_CONTEXT_DATA, * PNETUIO_FILE_CONTEXT_DATA;


/**
 * This macro will generate an inline function which will be used to get
 * a pointer to the file object's context memory in a type safe manner.
 */
WDF_DECLARE_CONTEXT_TYPE_WITH_NAME(NETUIO_FILE_CONTEXT_DATA, netuio_get_file_object_context_data)

/**
 * Functions to initialize the device and its callbacks
 */
NTSTATUS netuio_create_device(_Inout_ PWDFDEVICE_INIT DeviceInit);
NTSTATUS netuio_map_hw_resources(_In_ WDFDEVICE Device, _In_ WDFCMRESLIST Resources, _In_ WDFCMRESLIST ResourcesTranslated);
VOID netuio_free_hw_resources(_In_ WDFDEVICE Device);

#endif // NETUIO_DEV_H
