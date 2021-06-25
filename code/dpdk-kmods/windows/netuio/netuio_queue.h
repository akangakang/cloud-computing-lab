/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

#ifndef NETUIO_QUEUE_H
#define NETUIO_QUEUE_H

VOID
netuio_unmap_address_from_user_process(_In_ PNETUIO_CONTEXT_DATA netuio_contextdata);

NTSTATUS
netuio_queue_initialize(_In_ WDFDEVICE hDevice);

/**
 * Events from the IoQueue object
 */
EVT_WDF_IO_QUEUE_IO_DEVICE_CONTROL netuio_evt_IO_device_control;

EVT_WDF_IO_IN_CALLER_CONTEXT netuio_evt_IO_in_caller_context;

#endif // NETUIO_QUEUE_H
