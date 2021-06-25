/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

#ifndef NETUIO_DRV_H
#define NETUIO_DRV_H

#define INITGUID

#include <ntddk.h>
#include <wdf.h>

#include "netuio_dev.h"
#include "netuio_queue.h"

/**
 * Print output constants
 */
#define DPFLTR_NETUIO_INFO_LEVEL   35

/**
 * WDFDRIVER Events
 */
DRIVER_INITIALIZE DriverEntry;
EVT_WDF_DRIVER_DEVICE_ADD       netuio_evt_device_add;
EVT_WDF_DEVICE_PREPARE_HARDWARE netuio_evt_prepare_hw;
EVT_WDF_DEVICE_RELEASE_HARDWARE netuio_evt_release_hw;
EVT_WDF_FILE_CLOSE              netuio_evt_file_cleanup;

#endif // NETUIO_DRV_H
