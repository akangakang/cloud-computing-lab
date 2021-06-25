Developing Windows Drivers
==========================

Prerequisites
-------------

Building Windows Drivers is only possible on Windows.

1. Visual Studio 2019 Community or Professional Edition
2. Windows Driver Kit (WDK) for Windows 10, version 1903

Follow the official instructions to obtain all of the above:
https://docs.microsoft.com/en-us/windows-hardware/drivers/download-the-wdk


Build the Drivers
-----------------

Build from Visual Studio
~~~~~~~~~~~~~~~~~~~~~~~~

Open a solution (``*.sln``) with Visual Studio and build it (Ctrl+Shift+B).


Build from Command-Line
~~~~~~~~~~~~~~~~~~~~~~~

Run *Developer Command Prompt for VS 2019* from the Start menu.

Navigate to the solution directory (with ``*.sln``), then run:

.. code-block:: console

    msbuild

To build a particular combination of configuration and platform:

.. code-block:: console

    msbuild -p:Configuration=Debug;Platform=x64


Install the Drivers
-------------------

Disable Driver Signature Enforcement
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

By default Windows prohibits installing and loading drivers without `digital
signature`_ obtained from Microsoft. For development signature enforcement may
be disabled as follows.

In Elevated Command Prompt (from this point, sufficient privileges are
assumed):

.. code-block:: console

    bcdedit -set loadoptions DISABLE_INTEGRITY_CHECKS
    bcdedit -set TESTSIGNING ON
    shutdown -r -t 0

Upon reboot, an overlay message should appear on the desktop informing
that Windows is in test mode, which means it allows loading unsigned drivers.

.. _digital signature: https://docs.microsoft.com/en-us/windows-hardware/drivers/install/driver-signing

Install, List, and Remove Drivers
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Driver package is by default located in a subdirectory of its source tree,
e.g. ``x64\Debug\virt2phys\virt2phys`` (note two levels of ``virt2phys``).

To install the driver and bind associated devices to it:

.. code-block:: console

    pnputil /add-driver x64\Debug\virt2phys\virt2phys\virt2phys.inf /install

A graphical confirmation to load an unsigned driver will still appear.

On Windows Server additional steps are required if the driver uses a custom
setup class:

1. From "Device Manager", "Action" menu, select "Add legacy hardware".
2. It will launch the "Add Hardware Wizard". Click "Next".
3. Select second option "Install the hardware that I manually select
   from a list (Advanced)".
4. On the next screen, locate the driver device class.
5. Select it, and click "Next".
6. The previously installed drivers will now be installed for
   the appropriate devices (software devices will be created).

To list installed drivers:

.. code-block:: console

    pnputil /enum-drivers

To remove the driver package and to uninstall its devices:

.. code-block:: console

    pnputil /delete-driver oem2.inf /uninstall
