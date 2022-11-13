package main

import (
	"fmt"

	vk "github.com/vulkan-go/vulkan"
	"github.com/xlab/tablewriter"
)

type VulkanDeviceInfo struct {
	gpuDevices []vk.PhysicalDevice

	instance vk.Instance
	surface  vk.Surface
	device   vk.Device
}

func (v *VulkanDeviceInfo) Destroy() {
	if v == nil {
		return
	}
	v.gpuDevices = nil
	vk.DestroyDevice(v.device, nil)
	vk.DestroyInstance(v.instance, nil)
}

var appInfo = &vk.ApplicationInfo{
	SType:              vk.StructureTypeApplicationInfo,
	ApiVersion:         vk.MakeVersion(1, 0, 0),
	ApplicationVersion: vk.MakeVersion(1, 0, 0),
	PApplicationName:   "VulkanDevice\x00",
	PEngineName:        "vulkango.com\x00",
}

func NewVulkanDevice(appInfo *vk.ApplicationInfo, window uintptr) (*VulkanDeviceInfo, error) {
	v := &VulkanDeviceInfo{}

	// step 1: create a Vulkan instance.
	var instanceExtensions []string
	instanceCreateInfo := &vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        appInfo,
		EnabledExtensionCount:   uint32(len(instanceExtensions)),
		PpEnabledExtensionNames: instanceExtensions,
	}
	err := vk.Error(vk.CreateInstance(instanceCreateInfo, nil, &v.instance))
	if err != nil {
		err = fmt.Errorf("vkCreateInstance failed with %s", err)
		return nil, err
	} else {
		vk.InitInstance(v.instance)
	}

	if v.gpuDevices, err = getPhysicalDevices(v.instance); err != nil {
		v.gpuDevices = nil
		vk.DestroyInstance(v.instance, nil)
		return nil, err
	}

	// step 2: create a logical device from the first GPU available.
	queueCreateInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}
	deviceExtensions := []string{
		"VK_KHR_swapchain\x00",
	}
	deviceCreateInfo := &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
		PQueueCreateInfos:       queueCreateInfos,
		EnabledExtensionCount:   uint32(len(deviceExtensions)),
		PpEnabledExtensionNames: deviceExtensions,
	}
	var device vk.Device
	err = vk.Error(vk.CreateDevice(v.gpuDevices[0], deviceCreateInfo, nil, &device))
	if err != nil {
		v.gpuDevices = nil
		vk.DestroySurface(v.instance, v.surface, nil)
		vk.DestroyInstance(v.instance, nil)
		err = fmt.Errorf("vkCreateDevice failed with %s", err)
		return nil, err
	} else {
		v.device = device
	}

	return v, nil
}

func PrintInfo(v *VulkanDeviceInfo) {
	var gpuProperties vk.PhysicalDeviceProperties
	vk.GetPhysicalDeviceProperties(v.gpuDevices[0], &gpuProperties)
	gpuProperties.Deref()

	table := tablewriter.CreateTable()
	table.UTF8Box()
	table.AddTitle(vk.ToString(gpuProperties.DeviceName[:]))
	table.AddRow("Physical Device Vendor", fmt.Sprintf("%x", gpuProperties.VendorID))
	if gpuProperties.DeviceType != vk.PhysicalDeviceTypeOther {
		table.AddRow("Physical Device Type", physicalDeviceType(gpuProperties.DeviceType))
	}
	table.AddRow("Physical GPUs", len(v.gpuDevices))
	table.AddRow("API Version", vk.Version(gpuProperties.ApiVersion))
	table.AddRow("API Version Supported", vk.Version(gpuProperties.ApiVersion))
	table.AddRow("Driver Version", vk.Version(gpuProperties.DriverVersion))

	fmt.Println("\n" + table.Render())
}

func main() {
	orPanic(vk.SetDefaultGetInstanceProcAddr())
	orPanic(vk.Init())
	vkDevice, err := NewVulkanDevice(appInfo, 0)
	orPanic(err)
	PrintInfo(vkDevice)

	vkDevice.Destroy()
}

func getPhysicalDevices(instance vk.Instance) ([]vk.PhysicalDevice, error) {
	var gpuCount uint32
	err := vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, nil))
	if err != nil {
		err = fmt.Errorf("vkEnumeratePhysicalDevices failed with %s", err)
		return nil, err
	}
	if gpuCount == 0 {
		err = fmt.Errorf("getPhysicalDevice: no GPUs found on the system")
		return nil, err
	}
	gpuList := make([]vk.PhysicalDevice, gpuCount)
	err = vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, gpuList))
	if err != nil {
		err = fmt.Errorf("vkEnumeratePhysicalDevices failed with %s", err)
		return nil, err
	}
	return gpuList, nil
}

func physicalDeviceType(dev vk.PhysicalDeviceType) string {
	switch dev {
	case vk.PhysicalDeviceTypeIntegratedGpu:
		return "Integrated GPU"
	case vk.PhysicalDeviceTypeDiscreteGpu:
		return "Discrete GPU"
	case vk.PhysicalDeviceTypeVirtualGpu:
		return "Virtual GPU"
	case vk.PhysicalDeviceTypeCpu:
		return "CPU"
	case vk.PhysicalDeviceTypeOther:
		return "Other"
	default:
		return "Unknown"
	}
}

func orPanic(err interface{}) {
	switch v := err.(type) {
	case error:
		if v != nil {
			panic(err)
		}
	case vk.Result:
		if err := vk.Error(v); err != nil {
			panic(err)
		}
	case bool:
		if !v {
			panic("condition failed: != true")
		}
	}
}
