..  SPDX-License-Identifier: BSD-3-Clause
    Copyright(c) 2020 Microsoft Corporation.

Compiling the NetUIO Driver from Source
=======================================

Operating System
~~~~~~~~~~~~~~~~

The NetUIO source has been validated against the following operating systems:

* Windows Server 2016
* Windows Server 2019

Hardware Requirements
~~~~~~~~~~~~~~~~~~~~~
The NetUIO driver has been validated using the following network adapters on the Windows platform:

*
*

Software Requirements
~~~~~~~~~~~~~~~~~~~~~
For a list of required software tools please see the Prerequisites section in windows/README.rst.

Building the NetUIO Driver
--------------------------

Follow the steps below to build the NetUIO driver and install the driver for the network adapter.

* Clone the dpdk-kmods repository: git clone git://dpdk.org/dpdk-kmods
* Navigate to \dpdk-kmods\windows\netuio
* Load netuio.sln in Microsoft Visual Studio 2017 or 2019
* Choose Release as the configuration mode and Build the solution
* The resultant output files can be found in x64\Release\netuio

Installing netuio.sys Driver for development
--------------------------------------------
.. note::

   In a development environment, NetUIO driver can be loaded by enabling test-signing.
   Please implement adequate precautionary measures before installing a test-signed driver, replacing a signed-driver.

To ensure test-signed kernel-mode drivers can load on Windows, enable test-signing, using the following BCDEdit command.

C:\windows\system32>Bcdedit.exe -set TESTSIGNING ON

Windows displays the text "Test Mode" to remind users the system has test-signing enabled.
Refer to the MSDN documentation on how to Test-Sign a Driver Package.

To procure a WHQL signed NetUIO driver for Windows, please reach out to dpdkwin@microsoft.com

* Go to Device Manager -> Network Adapters.
* Right Click on target network adapter -> Select Update Driver.
* Select "Browse my computer for driver software".
* In the resultant window, select "Let me pick from a list of available drivers on my computer".
* Select "DPDK netUIO for Network Adapter" from the list of drivers.
* The NetUIO.sys driver is successfully installed.
