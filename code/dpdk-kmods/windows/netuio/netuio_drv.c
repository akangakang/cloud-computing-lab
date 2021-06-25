/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Microsoft Corporation.
 */

#include "netuio_drv.h"

#ifdef ALLOC_PRAGMA
#pragma alloc_text (INIT, DriverEntry)
#pragma alloc_text (PAGE, netuio_evt_device_add)
#endif

/*
Routine Description:
    DriverEntry initializes the driver and is the first routine called by the
    system after the driver is loaded. DriverEntry specifies the other entry
    points in the function driver, such as EvtDevice and DriverUnload.

Return Value:
    STATUS_SUCCESS if successful,
    STATUS_UNSUCCESSFUL otherwise.
 */
NTSTATUS
DriverEntry(_In_ PDRIVER_OBJECT DriverObject, _In_ PUNICODE_STRING RegistryPath)
{
    WDF_DRIVER_CONFIG config;
    NTSTATUS status;
    WDF_OBJECT_ATTRIBUTES attributes;

    WDF_OBJECT_ATTRIBUTES_INIT(&attributes);
    WDF_DRIVER_CONFIG_INIT(&config, netuio_evt_device_add);

    status = WdfDriverCreate(DriverObject, RegistryPath,
                             &attributes, &config,
                             WDF_NO_HANDLE);

    if (!NT_SUCCESS(status)) {
        return status;
    }

    return status;
}

/*
Routine Description:
    netuio_evt_device_add is called by the framework in response to AddDevice
    call from the PnP manager. We create and initialize a device object to
    represent a new instance of the device.

Return Value:
    NTSTATUS
 */
_Use_decl_annotations_
NTSTATUS
netuio_evt_device_add(WDFDRIVER Driver, PWDFDEVICE_INIT DeviceInit)
{
    UNREFERENCED_PARAMETER(Driver);
    return netuio_create_device(DeviceInit);
}

/*
Routine Description :
    Maps HW resources and retrieves the PCI BAR address(es) of the device

Return Value :
    STATUS_SUCCESS is successful.
    STATUS_<ERROR> otherwise
-*/
_Use_decl_annotations_
NTSTATUS
netuio_evt_prepare_hw(WDFDEVICE Device, WDFCMRESLIST Resources, WDFCMRESLIST ResourcesTranslated)
{
    NTSTATUS status;

    status = netuio_map_hw_resources(Device, Resources, ResourcesTranslated);
    if (!NT_SUCCESS(status)) {
        return status;
    }

    PNETUIO_CONTEXT_DATA  ctx = netuio_get_context_data(Device);
    DbgPrintEx(DPFLTR_IHVNETWORK_ID, DPFLTR_NETUIO_INFO_LEVEL, "netUIO Driver loaded...on device (B:D:F) %04d:%02d:%02d\n",
        ctx->addr.bus_num, ctx->addr.dev_num, ctx->addr.func_num);

    return status;
}

/*
Routine Description :
    Releases the resource mapped by netuio_evt_prepare_hw

Return Value :
    STATUS_SUCCESS always.
-*/
_Use_decl_annotations_
NTSTATUS
netuio_evt_release_hw(WDFDEVICE Device, WDFCMRESLIST ResourcesTranslated)
{
    UNREFERENCED_PARAMETER(ResourcesTranslated);

    netuio_free_hw_resources(Device);

    return STATUS_SUCCESS;
}

/*
Routine Description :
    EVT_WDF_FILE_CLEANUP callback, called when user process handle is closed.

    Undoes IOCTL_NETUIO_MAP_HW_INTO_USERSPACE.

Return value :
    None
-*/
_Use_decl_annotations_
VOID
netuio_evt_file_cleanup(WDFFILEOBJECT FileObject)
{
    PNETUIO_FILE_CONTEXT_DATA file_ctx;

    file_ctx = netuio_get_file_object_context_data(FileObject);

    if (file_ctx->bMapped == TRUE)
    {
        WDFDEVICE device;
        PNETUIO_CONTEXT_DATA ctx;

        device = WdfFileObjectGetDevice(FileObject);
        ctx = netuio_get_context_data(device);
        netuio_unmap_address_from_user_process(ctx);
        file_ctx->bMapped = FALSE;
    }
}
