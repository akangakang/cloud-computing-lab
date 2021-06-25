/* SPDX-License-Identifier: BSD-3-Clause
 * Copyright(c) 2020 Dmitry Kozlyuk
 */

#include <ntddk.h>
#include <wdf.h>
#include <wdmsec.h>
#include <initguid.h>

#include "virt2phys.h"

DRIVER_INITIALIZE DriverEntry;
EVT_WDF_DRIVER_DEVICE_ADD virt2phys_driver_EvtDeviceAdd;
EVT_WDF_IO_IN_CALLER_CONTEXT virt2phys_device_EvtIoInCallerContext;

NTSTATUS
DriverEntry(
	IN PDRIVER_OBJECT driver_object, IN PUNICODE_STRING registry_path)
{
	WDF_DRIVER_CONFIG config;
	WDF_OBJECT_ATTRIBUTES attributes;
	NTSTATUS status;

	PAGED_CODE();

	WDF_DRIVER_CONFIG_INIT(&config, virt2phys_driver_EvtDeviceAdd);
	WDF_OBJECT_ATTRIBUTES_INIT(&attributes);
	status = WdfDriverCreate(
			driver_object, registry_path,
			&attributes, &config, WDF_NO_HANDLE);
	if (!NT_SUCCESS(status)) {
		KdPrint(("WdfDriverCreate() failed, status=%08x\n", status));
	}

	return status;
}

_Use_decl_annotations_
NTSTATUS
virt2phys_driver_EvtDeviceAdd(
	WDFDRIVER driver, PWDFDEVICE_INIT init)
{
	WDF_OBJECT_ATTRIBUTES attributes;
	WDFDEVICE device;
	NTSTATUS status;

	UNREFERENCED_PARAMETER(driver);

	PAGED_CODE();

	WdfDeviceInitSetIoType(
		init, WdfDeviceIoNeither);
	WdfDeviceInitSetIoInCallerContextCallback(
		init, virt2phys_device_EvtIoInCallerContext);

	WDF_OBJECT_ATTRIBUTES_INIT(&attributes);

	status = WdfDeviceCreate(&init, &attributes, &device);
	if (!NT_SUCCESS(status)) {
		KdPrint(("WdfDeviceCreate() failed, status=%08x\n", status));
		return status;
	}

	status = WdfDeviceCreateDeviceInterface(
			device, &GUID_DEVINTERFACE_VIRT2PHYS, NULL);
	if (!NT_SUCCESS(status)) {
		KdPrint(("WdfDeviceCreateDeviceInterface() failed, "
			"status=%08x\n", status));
		return status;
	}

	return STATUS_SUCCESS;
}

_Use_decl_annotations_
VOID
virt2phys_device_EvtIoInCallerContext(
	IN WDFDEVICE device, IN WDFREQUEST request)
{
	WDF_REQUEST_PARAMETERS params;
	ULONG code;
	PVOID *virt;
	PHYSICAL_ADDRESS *phys;
	size_t size;
	NTSTATUS status;

	UNREFERENCED_PARAMETER(device);

	PAGED_CODE();

	WDF_REQUEST_PARAMETERS_INIT(&params);
	WdfRequestGetParameters(request, &params);

	if (params.Type != WdfRequestTypeDeviceControl) {
		KdPrint(("bogus request type=%u\n", params.Type));
		WdfRequestComplete(request, STATUS_NOT_SUPPORTED);
		return;
	}

	code = params.Parameters.DeviceIoControl.IoControlCode;
	if (code != IOCTL_VIRT2PHYS_TRANSLATE) {
		KdPrint(("bogus IO control code=%lu\n", code));
		WdfRequestComplete(request, STATUS_NOT_SUPPORTED);
		return;
	}

	status = WdfRequestRetrieveInputBuffer(
			request, sizeof(*virt), (PVOID *)&virt, &size);
	if (!NT_SUCCESS(status)) {
		KdPrint(("WdfRequestRetrieveInputBuffer() failed, "
			"status=%08x\n", status));
		WdfRequestComplete(request, status);
		return;
	}

	status = WdfRequestRetrieveOutputBuffer(
		request, sizeof(*phys), (PVOID *)&phys, &size);
	if (!NT_SUCCESS(status)) {
		KdPrint(("WdfRequestRetrieveOutputBuffer() failed, "
			"status=%08x\n", status));
		WdfRequestComplete(request, status);
		return;
	}

	*phys = MmGetPhysicalAddress(*virt);

	WdfRequestCompleteWithInformation(
		request, STATUS_SUCCESS, sizeof(*phys));
}
