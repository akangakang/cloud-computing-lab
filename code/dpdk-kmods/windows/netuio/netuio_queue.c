/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

#include "netuio_drv.h"

#ifdef ALLOC_PRAGMA
#pragma alloc_text (PAGE, netuio_queue_initialize)
#endif

static
BOOLEAN
netuio_get_usermode_mapping_flag(WDFREQUEST Request)
{
    WDFFILEOBJECT file_object;

    file_object = WdfRequestGetFileObject(Request);
    if (file_object == NULL)
    {
        goto end;
    }

    PNETUIO_FILE_CONTEXT_DATA file_ctx;

    file_ctx = netuio_get_file_object_context_data(file_object);
    if (file_ctx->bMapped)
    {
        return TRUE;
    }

end:
    return FALSE;
}

static
VOID
netuio_set_usermode_mapping_flag(WDFREQUEST Request)
{
    WDFFILEOBJECT file_object;

    file_object = WdfRequestGetFileObject(Request);
    if (file_object == NULL)
    {
        return;
    }

    PNETUIO_FILE_CONTEXT_DATA file_ctx;

    file_ctx = netuio_get_file_object_context_data(file_object);

    file_ctx->bMapped = TRUE;
}

static void
netuio_handle_get_hw_data_request(_In_ PNETUIO_CONTEXT_DATA ctx,
                                  _In_ PVOID outputBuf, _In_ size_t outputBufSize)
{
    ASSERT(outputBufSize == sizeof(struct device_info));

    struct device_info *dpdk_pvt_info = (struct device_info *)outputBuf;
    RtlZeroMemory(dpdk_pvt_info, outputBufSize);

    for (ULONG idx = 0; idx < PCI_MAX_BAR; idx++) {
        dpdk_pvt_info->hw[idx].phys_addr.QuadPart = ctx->bar[idx].base_addr.QuadPart;
        dpdk_pvt_info->hw[idx].user_mapped_virt_addr = ctx->dpdk_hw[idx].mem.user_mapped_virt_addr;
        dpdk_pvt_info->hw[idx].size = ctx->bar[idx].size;
    }
}

/*
Routine Description:
    Maps address ranges into the usermode process's address space.  The following
    ranges are mapped:

        * Any PCI BARs that our device was assigned
        * The scratch buffer of contiguous pages

Return Value:
    NTSTATUS
*/
static NTSTATUS
netuio_map_address_into_user_process(_In_ PNETUIO_CONTEXT_DATA ctx, WDFREQUEST Request)
{
    NTSTATUS status = STATUS_SUCCESS;

    if (netuio_get_usermode_mapping_flag(Request))
    {
        goto end;
    }

    // Map any device BAR(s) to the user's process context
    for (INT idx = 0; idx < PCI_MAX_BAR; idx++) {
        if (ctx->dpdk_hw[idx].mdl == NULL) {
            continue;
        }

        MmBuildMdlForNonPagedPool(ctx->dpdk_hw[idx].mdl);
        __try {
            ctx->dpdk_hw[idx].mem.user_mapped_virt_addr =
                MmMapLockedPagesSpecifyCache(ctx->dpdk_hw[idx].mdl, UserMode,
                                             MmCached, NULL, FALSE, NormalPagePriority);

            if (ctx->dpdk_hw[idx].mem.user_mapped_virt_addr == NULL) {
                status = STATUS_INSUFFICIENT_RESOURCES;
                goto end;
            }
        }
        __except (EXCEPTION_EXECUTE_HANDLER) {
            status = GetExceptionCode();
            goto end;
        }
    }

end:
    if (status != STATUS_SUCCESS) {
        netuio_unmap_address_from_user_process(ctx);
    }

    return status;
}

/*
Routine Description:
    Unmaps all address ranges from the usermode process address space.
    MUST be called in the context of the same process which created
    the mapping.

Return Value:
    None
 */
_Use_decl_annotations_
VOID
netuio_unmap_address_from_user_process(PNETUIO_CONTEXT_DATA ctx)
{
    for (INT idx = 0; idx < PCI_MAX_BAR; idx++) {
        if (ctx->dpdk_hw[idx].mem.user_mapped_virt_addr != NULL) {
            MmUnmapLockedPages(ctx->dpdk_hw[idx].mem.user_mapped_virt_addr,
                               ctx->dpdk_hw[idx].mdl);

            ctx->dpdk_hw[idx].mem.user_mapped_virt_addr = NULL;
        }
    }
}

/*
Routine Description:
    The I/O dispatch callbacks for the frameworks device object are configured here.
    A single default I/O Queue is configured for parallel request processing, and a
    driver context memory allocation is created to hold our structure QUEUE_CONTEXT.

Return Value:
    None
 */
_Use_decl_annotations_
NTSTATUS
netuio_queue_initialize(WDFDEVICE Device)
{
    WDFQUEUE queue;
    NTSTATUS status;
    WDF_IO_QUEUE_CONFIG    queueConfig;

    PAGED_CODE();

    // Configure a default queue so that requests that are not
    // configure-fowarded using WdfDeviceConfigureRequestDispatching to goto
    // other queues get dispatched here.
    WDF_IO_QUEUE_CONFIG_INIT_DEFAULT_QUEUE(&queueConfig, WdfIoQueueDispatchParallel);

    queueConfig.EvtIoDeviceControl = netuio_evt_IO_device_control;

    status = WdfIoQueueCreate(Device,
                              &queueConfig,
                              WDF_NO_OBJECT_ATTRIBUTES,
                              &queue);

    if( !NT_SUCCESS(status) ) {
        return status;
    }

    return status;
}

/*
Routine Description:
    This routine is invoked to preprocess an I/O request before being placed into a queue.
    It is guaranteed that it executes in the context of the process that generated the request.

Return Value:
    None
 */
_Use_decl_annotations_
VOID
netuio_evt_IO_in_caller_context(WDFDEVICE  Device,
                                WDFREQUEST Request)
{
    WDF_REQUEST_PARAMETERS params = { 0 };
    NTSTATUS status = STATUS_SUCCESS;
    PVOID    output_buf = NULL;
    size_t   output_buf_size;
    size_t  bytes_returned = 0;
    PNETUIO_CONTEXT_DATA  ctx = NULL;

    ctx = netuio_get_context_data(Device);

    WDF_REQUEST_PARAMETERS_INIT(&params);
    WdfRequestGetParameters(Request, &params);

    if (params.Type != WdfRequestTypeDeviceControl)
    {
        status = STATUS_INVALID_DEVICE_REQUEST;
        goto end;
    }

    // We only need to be in the context of the process that initiated the request
    //when we need to map memory to userspace. Otherwise, send the request back to framework.
    if (params.Parameters.DeviceIoControl.IoControlCode != IOCTL_NETUIO_MAP_HW_INTO_USERSPACE)
    {
        status = WdfDeviceEnqueueRequest(Device, Request);
        if (!NT_SUCCESS(status))
        {
            goto end;
        }
        return;
    }

    // Return relevant data to the caller
    status = WdfRequestRetrieveOutputBuffer(Request, sizeof(struct device_info), &output_buf, &output_buf_size);
    if (!NT_SUCCESS(status)) {
        goto end;
    }

    status = netuio_map_address_into_user_process(ctx, Request);
    if (!NT_SUCCESS(status)) {
        goto end;
    }

    netuio_set_usermode_mapping_flag(Request);

    netuio_handle_get_hw_data_request(ctx, output_buf, output_buf_size);
    bytes_returned = output_buf_size;

end:
    WdfRequestCompleteWithInformation(Request, status, bytes_returned);

    return;
}

/*
Routine Description:
    This event is invoked when the framework receives IRP_MJ_DEVICE_CONTROL request.

Return Value:
    None
 */
_Use_decl_annotations_
VOID
netuio_evt_IO_device_control(WDFQUEUE Queue, WDFREQUEST Request,
                             size_t OutputBufferLength, size_t InputBufferLength,
                             ULONG IoControlCode)
{
    UNREFERENCED_PARAMETER(OutputBufferLength);
    UNREFERENCED_PARAMETER(InputBufferLength);

    NTSTATUS status = STATUS_SUCCESS;
    PVOID    input_buf = NULL, output_buf = NULL;
    size_t   input_buf_size, output_buf_size;
    size_t  bytes_returned = 0;

    WDFDEVICE device = WdfIoQueueGetDevice(Queue);

    PNETUIO_CONTEXT_DATA  ctx;
    ctx = netuio_get_context_data(device);

    if (IoControlCode != IOCTL_NETUIO_PCI_CONFIG_IO)
    {
        status = STATUS_INVALID_DEVICE_REQUEST;
        goto end;
    }

    // First retrieve the input buffer and see if it matches our device
    status = WdfRequestRetrieveInputBuffer(Request, sizeof(struct dpdk_pci_config_io), &input_buf, &input_buf_size);
    if (!NT_SUCCESS(status)) {
        goto end;
    }

    struct dpdk_pci_config_io *dpdk_pci_io_input = (struct dpdk_pci_config_io *)input_buf;

    if (dpdk_pci_io_input->access_size != 1 &&
        dpdk_pci_io_input->access_size != 2 &&
        dpdk_pci_io_input->access_size != 4 &&
        dpdk_pci_io_input->access_size != 8) {
        status = STATUS_INVALID_PARAMETER;
        goto end;
    }

    // Retrieve output buffer
    status = WdfRequestRetrieveOutputBuffer(Request, sizeof(UINT64), &output_buf, &output_buf_size);
    if (!NT_SUCCESS(status)) {
        goto end;
    }
    ASSERT(output_buf_size == OutputBufferLength);

    if (dpdk_pci_io_input->op == PCI_IO_READ) {
        *(UINT64 *)output_buf = 0;
        bytes_returned = ctx->bus_interface.GetBusData(
            ctx->bus_interface.Context,
            PCI_WHICHSPACE_CONFIG,
            output_buf,
            dpdk_pci_io_input->offset,
            dpdk_pci_io_input->access_size);
    }
    else if (dpdk_pci_io_input->op == PCI_IO_WRITE) {
        // returns bytes written
        bytes_returned = ctx->bus_interface.SetBusData(
            ctx->bus_interface.Context,
            PCI_WHICHSPACE_CONFIG,
            (PVOID)&dpdk_pci_io_input->data,
            dpdk_pci_io_input->offset,
            dpdk_pci_io_input->access_size);
    }
    else {
        status = STATUS_INVALID_PARAMETER;
        goto end;
    }

    if (bytes_returned != dpdk_pci_io_input->access_size) {
        status = STATUS_INVALID_PARAMETER;
    }

end:
    WdfRequestCompleteWithInformation(Request, status, bytes_returned);

    return;
}
