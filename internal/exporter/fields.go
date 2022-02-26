package exporter

import (
	"bufio"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// qField stands for query field - the field name before the query
type qField string

// rField stands for returned field - the field name as returned by the lynxi-smi
type rField string
type Version int64
type PciInfo int64

const (
	__COLON_SEP__     = ":"
	__LINE_FEED_STR__ = "\n"
	__LINE_FEED_SEP__ = '\n'
	__PER_SEP__       = "%"
	__V_SEP__         = "V"
	__C_SEP__         = "C"
	__SPCAE_SEP__     = " "
	__W_SEP__         = "W"
	__DOT_SEP__       = "."
	__COMMA_SEP__     = ","
)

const (
	SDK Version = iota
	Driver
	SMI
)

const (
	DefaultLynSmiCommand        = "lynxi-smi"
	DefaultFindLynDriverCommand = "dpkg"
	DefaultPCICommand           = "lspci"
	LynDriverStr                = "lyndriver"
	LynSdkStr                   = "lynsdk"
	VersionShortStr             = "V"
)

const (
	VendorId PciInfo = iota
	DeviceId
	SubVendorId
	SubDeviceId
	BusNum
	Device
	Function
	MaxSpeed
	MaxWidth
	CurrentSpeed
	CurrentWidth
	NumaNodeId
	NumaCPUList
)

var (
	fallbackQFieldToRFieldMap = map[qField]rField{
		__TIMESTAMP_KEY__:                          "timestamp",
		__BOARD_INDEX_KEY__:                        "board_index",
		__PRODUCT_NAME_KEY__:                       "name",
		__PRODUCT_NUMBER_KEY__:                     "product_number",
		__DRIVER_VERSION_KEY__:                     "driver_version",
		__FIRMWARE_VERSION_KEY__:                   "firmware_version",
		__SERIAL_NUMBER_KEY__:                      "serial_number",
		__CHIP_COUNT_KEY__:                         "chip_count",
		__CHIP_ID_CHIP0_KEY__:                      "chip_id.chip0",
		__CHIP_ID_CHIP1_KEY__:                      "chip_id.chip1",
		__CHIP_ID_CHIP2_KEY__:                      "chip_id.chip2",
		__UUID_CHIP0_KEY__:                         "uuid.chip0",
		__UUID_CHIP1_KEY__:                         "uuid.chip1",
		__UUID_CHIP2_KEY__:                         "uuid.chip2",
		__CHIP_INDEX_CHIP0_KEY__:                   "chip_index.chip0",
		__CHIP_INDEX_CHIP1_KEY__:                   "chip_index.chip1",
		__CHIP_INDEX_CHIP2_KEY__:                   "chip_index.chip2",
		__UTILIZATION_APU_TOTAL_KEY__:              "utilization.apu.total",
		__UTILIZATION_APU_CHIP0_KEY__:              "utilization.apu.chip0",
		__UTILIZATION_APU_CHIP1_KEY__:              "utilization.apu.chip1",
		__UTILIZATION_APU_CHIP2_KEY__:              "utilization.apu.chip2",
		__UTILIZATION_CPU_TOTAL_KEY__:              "utilization.cpu.total",
		__UTILIZATION_CPU_CHIP0_KEY__:              "utilization.cpu.chip0",
		__UTILIZATION_CPU_CHIP1_KEY__:              "utilization.cpu.chip1",
		__UTILIZATION_CPU_CHIP2_KEY__:              "utilization.cpu.chip2",
		__UTILIZATION_VIC_TOTAL_KEY__:              "utilization.vic.total",
		__UTILIZATION_VIC_CHIP0_KEY__:              "utilization.vic.chip0",
		__UTILIZATION_VIC_CHIP1_KEY__:              "utilization.vic.chip1",
		__UTILIZATION_VIC_CHIP2_KEY__:              "utilization.vic.chip2",
		__UTILIZATION_MEMORY_TOTAL_KEY__:           "utilization.memory.total",
		__UTILIZATION_MEMORY_CHIP0_KEY__:           "utilization.memory.chip0",
		__UTILIZATION_MEMORY_CHIP1_KEY__:           "utilization.memory.chip1",
		__UTILIZATION_MEMORY_CHIP2_KEY__:           "utilization.memory.chip2",
		__UTILIZATION_IPE_FPS_TOTAL_KEY__:          "utilization.ipeFps.total",
		__UTILIZATION_IPE_FPS_CHIP0_KEY__:          "utilization.ipeFps.chip0",
		__UTILIZATION_IPE_FPS_CHIP1_KEY__:          "utilization.ipeFps.chip1",
		__UTILIZATION_IPE_FPS_CHIP2_KEY__:          "utilization.ipeFps.chip2",
		__PCI_SUB_VENDOR_ID_CHIP0_KEY__:            "pci.sub_vendor_id.chip0",
		__PCI_SUB_VENDOR_ID_CHIP1_KEY__:            "pci.sub_vendor_id.chip1",
		__PCI_SUB_VENDOR_ID_CHIP2_KEY__:            "pci.sub_vendor_id.chip2",
		__PCI_VENDOR_ID_CHIP0_KEY__:                "pci.vendor_id.chip0",
		__PCI_VENDOR_ID_CHIP1_KEY__:                "pci.vendor_id.chip1",
		__PCI_VENDOR_ID_CHIP2_KEY__:                "pci.vendor_id.chip2",
		__PCI_BUS_CHIP0_KEY__:                      "pci.bus.chip0",
		__PCI_BUS_CHIP1_KEY__:                      "pci.bus.chip1",
		__PCI_BUS_CHIP2_KEY__:                      "pci.bus.chip2",
		__PCI_DEVICE_ID_CHIP0_KEY__:                "pci.device_id.chip0",
		__PCI_DEVICE_ID_CHIP1_KEY__:                "pci.device_id.chip1",
		__PCI_DEVICE_ID_CHIP2_KEY__:                "pci.device_id.chip2",
		__PCI_SUB_DEVICE_ID_CHIP0_KEY__:            "pci.sub_device_id.chip0",
		__PCI_SUB_DEVICE_ID_CHIP1_KEY__:            "pci.sub_device_id.chip1",
		__PCI_SUB_DEVICE_ID_CHIP2_KEY__:            "pci.sub_device_id.chip2",
		__PCI_DEVICE_CHIP0_KEY__:                   "pci.device.chip0",
		__PCI_DEVICE_CHIP1_KEY__:                   "pci.device.chip1",
		__PCI_DEVICE_CHIP2_KEY__:                   "pci.device.chip2",
		__PCI_FUNCTION_CHIP0_KEY__:                 "pci.function.chip0",
		__PCI_FUNCTION_CHIP1_KEY__:                 "pci.function.chip1",
		__PCI_FUNCTION_CHIP2_KEY__:                 "pci.function.chip2",
		__PCI_NUMA_NODE_ID_CHIP0_KEY__:             "pci.numa.node_id.chip0",
		__PCI_NUMA_NODE_ID_CHIP1_KEY__:             "pci.numa.node_id.chip1",
		__PCI_NUMA_NODE_ID_CHIP2_KEY__:             "pci.numa.node_id.chip2",
		__PCI_NUMA_CPU_CHIP0_KEY__:                 "pci.numa.cpu.chip0",
		__PCI_NUMA_CPU_CHIP1_KEY__:                 "pci.numa.cpu.chip1",
		__PCI_NUMA_CPU_CHIP2_KEY__:                 "pci.numa.cpu.chip2",
		__PCIE_LINK_SPEED_MAX_CHIP0_KEY__:          "pcie.link.speed.max.chip0",
		__PCIE_LINK_SPEED_MAX_CHIP1_KEY__:          "pcie.link.speed.max.chip1",
		__PCIE_LINK_SPEED_MAX_CHIP2_KEY__:          "pcie.link.speed.max.chip2",
		__PCIE_LINK_SPEED_CURRENT_CHIP0_KEY__:      "pcie.link.speed.current.chip0",
		__PCIE_LINK_SPEED_CURRENT_CHIP1_KEY__:      "pcie.link.speed.current.chip1",
		__PCIE_LINK_SPEED_CURRENT_CHIP2_KEY__:      "pcie.link.speed.current.chip2",
		__PCIE_LINK_GEN_MAX_CHIP0_KEY__:            "pcie.link.gen.max.chip0",
		__PCIE_LINK_GEN_MAX_CHIP1_KEY__:            "pcie.link.gen.max.chip1",
		__PCIE_LINK_GEN_MAX_CHIP2_KEY__:            "pcie.link.gen.max.chip2",
		__PCIE_LINK_GEN_CURRENT_CHIP0_KEY__:        "pcie.link.gen.current.chip0",
		__PCIE_LINK_GEN_CURRENT_CHIP1_KEY__:        "pcie.link.gen.current.chip1",
		__PCIE_LINK_GEN_CURRENT_CHIP2_KEY__:        "pcie.link.gen.current.chip2",
		__FAN_SPEED_KEY__:                          "fan.speed",
		__TEMPERATURE_CURRENT_CHIP0_KEY__:          "temperature.current.chip0",
		__TEMPERATURE_CURRENT_CHIP1_KEY__:          "temperature.current.chip1",
		__TEMPERATURE_CURRENT_CHIP2_KEY__:          "temperature.current.chip2",
		__VOLTAGE_CURRENT_CHIP0_KEY__:              "voltage.current.chip0",
		__VOLTAGE_CURRENT_CHIP1_KEY__:              "voltage.current.chip1",
		__VOLTAGE_CURRENT_CHIP2_KEY__:              "voltage.current.chip2",
		__VOLTAGE_BOARD_INPUT_KEY__:                "voltage.board.input",
		__CLOCKS_CURRENT_APU_CHIP0_KEY__:           "clocks.current.apu.chip0",
		__CLOCKS_CURRENT_APU_CHIP1_KEY__:           "clocks.current.apu.chip1",
		__CLOCKS_CURRENT_APU_CHIP2_KEY__:           "clocks.current.apu.chip2",
		__CLOCKS_CURRENT_CPU_CHIP0_KEY__:           "clocks.current.cpu.chip0",
		__CLOCKS_CURRENT_CPU_CHIP1_KEY__:           "clocks.current.cpu.chip1",
		__CLOCKS_CURRENT_CPU_CHIP2_KEY__:           "clocks.current.cpu.chip2",
		__CLOCKS_CURRENT_MEMORY_CHIP0_KEY__:        "clocks.current.memory.chip0",
		__CLOCKS_CURRENT_MEMORY_CHIP1_KEY__:        "clocks.current.memory.chip1",
		__CLOCKS_CURRENT_MEMORY_CHIP2_KEY__:        "clocks.current.memory.chip2",
		__CLOCKS_MAX_APU_CHIP0_KEY__:               "clocks.current.apu.max.chip0",
		__CLOCKS_MAX_APU_CHIP1_KEY__:               "clocks.current.apu.max.chip1",
		__CLOCKS_MAX_APU_CHIP2_KEY__:               "clocks.current.apu.max.chip2",
		__CLOCKS_MAX_CPU_CHIP0_KEY__:               "clocks.current.cpu.max.chip0",
		__CLOCKS_MAX_CPU_CHIP1_KEY__:               "clocks.current.cpu.max.chip1",
		__CLOCKS_MAX_CPU_CHIP2_KEY__:               "clocks.current.cpu.max.chip2",
		__CLOCKS_MAX_MEMORY_CHIP0_KEY__:            "clocks.current.memory.max.chip0",
		__CLOCKS_MAX_MEMORY_CHIP1_KEY__:            "clocks.current.memory.max.chip1",
		__CLOCKS_MAX_MEMORY_CHIP2_KEY__:            "clocks.current.memory.max.chip2",
		__POWER_DRAW__:                             "power.draw",
		__POWER_LIMIT__:                            "power.limit",
		__ECC_MODE_CURRENT_CHIP0_KEY__:             "ecc.mode.current.chip0",
		__ECC_MODE_CURRENT_CHIP1_KEY__:             "ecc.mode.current.chip1",
		__ECC_MODE_CURRENT_CHIP2_KEY__:             "ecc.mode.current.chip2",
		__ECC_ERRORS_CORRECTED_TOTAL_KEY__:         "ecc.errors.corrected.total",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP0_KEY__:   "ecc.errors.corrected.total.chip0",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP1_KEY__:   "ecc.errors.corrected.total.chip1",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP2_KEY__:   "ecc.errors.corrected.total.chip2",
		__ECC_ERRORS_UNCORRECTED_TOTAL_KEY__:       "ecc.errors.uncorrected.total",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP0_KEY__: "ecc.errors.uncorrected.total.chip0",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP1_KEY__: "ecc.errors.uncorrected.total.chip1",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP2_KEY__: "ecc.errors.uncorrected.total.chip2",
	}
	fallbackQField = []qField{
		__TIMESTAMP_KEY__,
		__BOARD_INDEX_KEY__,
		__PRODUCT_NAME_KEY__,
		__PRODUCT_NUMBER_KEY__,
		__DRIVER_VERSION_KEY__,
		__FIRMWARE_VERSION_KEY__,
		__SERIAL_NUMBER_KEY__,
		__CHIP_COUNT_KEY__,
		__CHIP_ID_CHIP0_KEY__,
		__CHIP_ID_CHIP1_KEY__,
		__CHIP_ID_CHIP2_KEY__,
		__UUID_CHIP0_KEY__,
		__UUID_CHIP1_KEY__,
		__UUID_CHIP2_KEY__,
		__CHIP_INDEX_CHIP0_KEY__,
		__CHIP_INDEX_CHIP1_KEY__,
		__CHIP_INDEX_CHIP2_KEY__,
		__UTILIZATION_APU_TOTAL_KEY__,
		__UTILIZATION_APU_CHIP0_KEY__,
		__UTILIZATION_APU_CHIP1_KEY__,
		__UTILIZATION_APU_CHIP2_KEY__,
		__UTILIZATION_CPU_TOTAL_KEY__,
		__UTILIZATION_CPU_CHIP0_KEY__,
		__UTILIZATION_CPU_CHIP1_KEY__,
		__UTILIZATION_CPU_CHIP2_KEY__,
		__UTILIZATION_VIC_TOTAL_KEY__,
		__UTILIZATION_VIC_CHIP0_KEY__,
		__UTILIZATION_VIC_CHIP1_KEY__,
		__UTILIZATION_VIC_CHIP2_KEY__,
		__UTILIZATION_MEMORY_TOTAL_KEY__,
		__UTILIZATION_MEMORY_CHIP0_KEY__,
		__UTILIZATION_MEMORY_CHIP1_KEY__,
		__UTILIZATION_MEMORY_CHIP2_KEY__,
		__UTILIZATION_IPE_FPS_TOTAL_KEY__,
		__UTILIZATION_IPE_FPS_CHIP0_KEY__,
		__UTILIZATION_IPE_FPS_CHIP1_KEY__,
		__UTILIZATION_IPE_FPS_CHIP2_KEY__,
		__PCI_SUB_VENDOR_ID_CHIP0_KEY__,
		__PCI_SUB_VENDOR_ID_CHIP1_KEY__,
		__PCI_SUB_VENDOR_ID_CHIP2_KEY__,
		__PCI_VENDOR_ID_CHIP0_KEY__,
		__PCI_VENDOR_ID_CHIP1_KEY__,
		__PCI_VENDOR_ID_CHIP2_KEY__,
		__PCI_BUS_CHIP0_KEY__,
		__PCI_BUS_CHIP1_KEY__,
		__PCI_BUS_CHIP2_KEY__,
		__PCI_DEVICE_ID_CHIP0_KEY__,
		__PCI_DEVICE_ID_CHIP1_KEY__,
		__PCI_DEVICE_ID_CHIP2_KEY__,
		__PCI_SUB_DEVICE_ID_CHIP0_KEY__,
		__PCI_SUB_DEVICE_ID_CHIP1_KEY__,
		__PCI_SUB_DEVICE_ID_CHIP2_KEY__,
		__PCI_DEVICE_CHIP0_KEY__,
		__PCI_DEVICE_CHIP1_KEY__,
		__PCI_DEVICE_CHIP2_KEY__,
		__PCI_FUNCTION_CHIP0_KEY__,
		__PCI_FUNCTION_CHIP1_KEY__,
		__PCI_FUNCTION_CHIP2_KEY__,
		__PCI_NUMA_NODE_ID_CHIP0_KEY__,
		__PCI_NUMA_NODE_ID_CHIP1_KEY__,
		__PCI_NUMA_NODE_ID_CHIP2_KEY__,
		__PCI_NUMA_CPU_CHIP0_KEY__,
		__PCI_NUMA_CPU_CHIP1_KEY__,
		__PCI_NUMA_CPU_CHIP2_KEY__,
		__PCIE_LINK_SPEED_MAX_CHIP0_KEY__,
		__PCIE_LINK_SPEED_MAX_CHIP1_KEY__,
		__PCIE_LINK_SPEED_MAX_CHIP2_KEY__,
		__PCIE_LINK_SPEED_CURRENT_CHIP0_KEY__,
		__PCIE_LINK_SPEED_CURRENT_CHIP1_KEY__,
		__PCIE_LINK_SPEED_CURRENT_CHIP2_KEY__,
		__PCIE_LINK_GEN_MAX_CHIP0_KEY__,
		__PCIE_LINK_GEN_MAX_CHIP1_KEY__,
		__PCIE_LINK_GEN_MAX_CHIP2_KEY__,
		__PCIE_LINK_GEN_CURRENT_CHIP0_KEY__,
		__PCIE_LINK_GEN_CURRENT_CHIP1_KEY__,
		__PCIE_LINK_GEN_CURRENT_CHIP2_KEY__,
		__FAN_SPEED_KEY__,
		__TEMPERATURE_CURRENT_CHIP0_KEY__,
		__TEMPERATURE_CURRENT_CHIP1_KEY__,
		__TEMPERATURE_CURRENT_CHIP2_KEY__,
		__VOLTAGE_CURRENT_CHIP0_KEY__,
		__VOLTAGE_CURRENT_CHIP1_KEY__,
		__VOLTAGE_CURRENT_CHIP2_KEY__,
		__VOLTAGE_BOARD_INPUT_KEY__,
		__CLOCKS_CURRENT_APU_CHIP0_KEY__,
		__CLOCKS_CURRENT_APU_CHIP1_KEY__,
		__CLOCKS_CURRENT_APU_CHIP2_KEY__,
		__CLOCKS_CURRENT_CPU_CHIP0_KEY__,
		__CLOCKS_CURRENT_CPU_CHIP1_KEY__,
		__CLOCKS_CURRENT_CPU_CHIP2_KEY__,
		__CLOCKS_CURRENT_MEMORY_CHIP0_KEY__,
		__CLOCKS_CURRENT_MEMORY_CHIP1_KEY__,
		__CLOCKS_CURRENT_MEMORY_CHIP2_KEY__,
		__CLOCKS_MAX_APU_CHIP0_KEY__,
		__CLOCKS_MAX_APU_CHIP1_KEY__,
		__CLOCKS_MAX_APU_CHIP2_KEY__,
		__CLOCKS_MAX_CPU_CHIP0_KEY__,
		__CLOCKS_MAX_CPU_CHIP1_KEY__,
		__CLOCKS_MAX_CPU_CHIP2_KEY__,
		__CLOCKS_MAX_MEMORY_CHIP0_KEY__,
		__CLOCKS_MAX_MEMORY_CHIP1_KEY__,
		__CLOCKS_MAX_MEMORY_CHIP2_KEY__,
		__POWER_DRAW__,
		__POWER_LIMIT__,
		__ECC_MODE_CURRENT_CHIP0_KEY__,
		__ECC_MODE_CURRENT_CHIP1_KEY__,
		__ECC_MODE_CURRENT_CHIP2_KEY__,
		__ECC_ERRORS_CORRECTED_TOTAL_KEY__,
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP0_KEY__,
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP1_KEY__,
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP2_KEY__,
		__ECC_ERRORS_UNCORRECTED_TOTAL_KEY__,
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP0_KEY__,
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP1_KEY__,
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP2_KEY__,
	}
	fallbackQFieldComment = map[qField]rField{
		__TIMESTAMP_KEY__:                          "The timestamp of when the query was made in format timestamp.",
		__BOARD_INDEX_KEY__:                        "Zero based index of the APU board. Can change at each boot.",
		__PRODUCT_NAME_KEY__:                       "the product name of the APU board.",
		__PRODUCT_NUMBER_KEY__:                     "the product number of the APU board.",
		__DRIVER_VERSION_KEY__:                     "The version of the installed LYNXI display driver. This is an alphanumeric string.",
		__FIRMWARE_VERSION_KEY__:                   "The version of the installed LYNXI display firmware driver.",
		__SERIAL_NUMBER_KEY__:                      "This number matches the serial number physically printed on each board. It is a globally unique immutable alphanumeric value.",
		__CHIP_COUNT_KEY__:                         "The number of LYNXI APUs in the system.",
		__CHIP_ID_CHIP0_KEY__:                      "This value is the globally unique immutable alphanumeric identifier of the APU0",
		__CHIP_ID_CHIP1_KEY__:                      "This value is the globally unique immutable alphanumeric identifier of the APU1",
		__CHIP_ID_CHIP2_KEY__:                      "This value is the globally unique immutable alphanumeric identifier of the APU2",
		__UUID_CHIP0_KEY__:                         "This value is the globally unique immutable alphanumeric identifier of the APU0. It does not correspond to any physical label on the board.",
		__UUID_CHIP1_KEY__:                         "This value is the globally unique immutable alphanumeric identifier of the APU1. It does not correspond to any physical label on the board.",
		__UUID_CHIP2_KEY__:                         "This value is the globally unique immutable alphanumeric identifier of the APU2. It does not correspond to any physical label on the board.",
		__CHIP_INDEX_CHIP0_KEY__:                   "Zero based index of the APU0. Can change at each boot.",
		__CHIP_INDEX_CHIP1_KEY__:                   "Zero based index of the APU1. Can change at each boot.",
		__CHIP_INDEX_CHIP2_KEY__:                   "Zero based index of the APU2. Can change at each boot.",
		__UTILIZATION_APU_TOTAL_KEY__:              "Percent of time over the past sample period during which global (device) apu was being read or written.",
		__UTILIZATION_APU_CHIP0_KEY__:              "Percent of time over the past sample period during which global (APU0) apu was being read or written.",
		__UTILIZATION_APU_CHIP1_KEY__:              "Percent of time over the past sample period during which global (APU1) apu was being read or written.",
		__UTILIZATION_APU_CHIP2_KEY__:              "Percent of time over the past sample period during which global (APU2) apu was being read or written.",
		__UTILIZATION_CPU_TOTAL_KEY__:              "Percent of time over the past sample period during which global (device) cpu was being read or written.",
		__UTILIZATION_CPU_CHIP0_KEY__:              "Percent of time over the past sample period during which global (APU0) cpu was being read or written.",
		__UTILIZATION_CPU_CHIP1_KEY__:              "Percent of time over the past sample period during which global (APU1) cpu was being read or written.",
		__UTILIZATION_CPU_CHIP2_KEY__:              "Percent of time over the past sample period during which global (APU2) cpu was being read or written.",
		__UTILIZATION_VIC_TOTAL_KEY__:              "Percent of time over the past sample period during which global (device) vic was being read or written.",
		__UTILIZATION_VIC_CHIP0_KEY__:              "Percent of time over the past sample period during which global (APU0) vic was being read or written.",
		__UTILIZATION_VIC_CHIP1_KEY__:              "Percent of time over the past sample period during which global (APU1) vic was being read or written.",
		__UTILIZATION_VIC_CHIP2_KEY__:              "Percent of time over the past sample period during which global (APU2) vic was being read or written.",
		__UTILIZATION_MEMORY_TOTAL_KEY__:           "Percent of time over the past sample period during which global (device) memory was being read or written.",
		__UTILIZATION_MEMORY_CHIP0_KEY__:           "Percent of time over the past sample period during which global (APU0) memory was being read or written.",
		__UTILIZATION_MEMORY_CHIP1_KEY__:           "Percent of time over the past sample period during which global (APU1) memory was being read or written.",
		__UTILIZATION_MEMORY_CHIP2_KEY__:           "Percent of time over the past sample period during which global (APU2) memory was being read or written.",
		__UTILIZATION_IPE_FPS_TOTAL_KEY__:          "Percent of time over the past sample period during which global (device) ipe was being read or written.",
		__UTILIZATION_IPE_FPS_CHIP0_KEY__:          "Percent of time over the past sample period during which global (APU0) ipe was being read or written.",
		__UTILIZATION_IPE_FPS_CHIP1_KEY__:          "Percent of time over the past sample period during which global (APU1) ipe was being read or written.",
		__UTILIZATION_IPE_FPS_CHIP2_KEY__:          "Percent of time over the past sample period during which global (APU2) ipe was being read or written.",
		__PCI_SUB_VENDOR_ID_CHIP0_KEY__:            "pci.sub_vendor_id.chip0, in hex.",
		__PCI_SUB_VENDOR_ID_CHIP1_KEY__:            "pci.sub_vendor_id.chip1, in hex.",
		__PCI_SUB_VENDOR_ID_CHIP2_KEY__:            "pci.sub_vendor_id.chip2, in hex.",
		__PCI_VENDOR_ID_CHIP0_KEY__:                "pci.vendor_id.chip0, in hex.",
		__PCI_VENDOR_ID_CHIP1_KEY__:                "pci.vendor_id.chip1, in hex.",
		__PCI_VENDOR_ID_CHIP2_KEY__:                "pci.vendor_id.chip2, in hex.",
		__PCI_BUS_CHIP0_KEY__:                      "pci.bus.chip0 , in hex.",
		__PCI_BUS_CHIP1_KEY__:                      "pci.bus.chip1, in hex.",
		__PCI_BUS_CHIP2_KEY__:                      "pci.bus.chip2, in hex.",
		__PCI_DEVICE_ID_CHIP0_KEY__:                "pci.device_id.chip0, in hex.",
		__PCI_DEVICE_ID_CHIP1_KEY__:                "pci.device_id.chip1, in hex.",
		__PCI_DEVICE_ID_CHIP2_KEY__:                "pci.device_id.chip2, in hex.",
		__PCI_SUB_DEVICE_ID_CHIP0_KEY__:            "pci.sub_device_id.chip0, in hex.",
		__PCI_SUB_DEVICE_ID_CHIP1_KEY__:            "pci.sub_device_id.chip1, in hex.",
		__PCI_SUB_DEVICE_ID_CHIP2_KEY__:            "pci.sub_device_id.chip2, in hex.",
		__PCI_DEVICE_CHIP0_KEY__:                   "pci.device.chip0, in hex.",
		__PCI_DEVICE_CHIP1_KEY__:                   "pci.device.chip1, in hex.",
		__PCI_DEVICE_CHIP2_KEY__:                   "pci.device.chip2, in hex.",
		__PCI_FUNCTION_CHIP0_KEY__:                 "pci.function.chip0, in hex.",
		__PCI_FUNCTION_CHIP1_KEY__:                 "pci.function.chip1, in hex.",
		__PCI_FUNCTION_CHIP2_KEY__:                 "pci.function.chip2, in hex.",
		__PCI_NUMA_NODE_ID_CHIP0_KEY__:             "pci.numa.node_id.chip0.",
		__PCI_NUMA_NODE_ID_CHIP1_KEY__:             "pci.numa.node_id.chip1.",
		__PCI_NUMA_NODE_ID_CHIP2_KEY__:             "pci.numa.node_id.chip2.",
		__PCI_NUMA_CPU_CHIP0_KEY__:                 "pci.numa.cpu.chip0.",
		__PCI_NUMA_CPU_CHIP1_KEY__:                 "pci.numa.cpu.chip1.",
		__PCI_NUMA_CPU_CHIP2_KEY__:                 "pci.numa.cpu.chip2.",
		__PCIE_LINK_SPEED_MAX_CHIP0_KEY__:          "The maximum PCI-E link width possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_SPEED_MAX_CHIP1_KEY__:          "The maximum PCI-E link width possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_SPEED_MAX_CHIP2_KEY__:          "pThe maximum PCI-E link width possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_SPEED_CURRENT_CHIP0_KEY__:      "The current PCI-E link width. These may be reduced when the APU is not in use.",
		__PCIE_LINK_SPEED_CURRENT_CHIP1_KEY__:      "The current PCI-E link width. These may be reduced when the APU is not in use.",
		__PCIE_LINK_SPEED_CURRENT_CHIP2_KEY__:      "The current PCI-E link width. These may be reduced when the APU is not in use.",
		__PCIE_LINK_GEN_MAX_CHIP0_KEY__:            "The maximum PCI-E link generation possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_GEN_MAX_CHIP1_KEY__:            "The maximum PCI-E link generation possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_GEN_MAX_CHIP2_KEY__:            "The maximum PCI-E link generation possible with this APU and system configuration. For example, if the APU supports a higher PCIe generation than the system supports then this reports the system PCIe generation.",
		__PCIE_LINK_GEN_CURRENT_CHIP0_KEY__:        "The current PCI-E link generation. These may be reduced when the APU is not in use.",
		__PCIE_LINK_GEN_CURRENT_CHIP1_KEY__:        "The current PCI-E link generation. These may be reduced when the APU is not in use.",
		__PCIE_LINK_GEN_CURRENT_CHIP2_KEY__:        "The current PCI-E link generation. These may be reduced when the APU is not in use.",
		__FAN_SPEED_KEY__:                          "Fan speed, in %.",
		__TEMPERATURE_CURRENT_CHIP0_KEY__:          "Core APU0 temperature. in degrees C.",
		__TEMPERATURE_CURRENT_CHIP1_KEY__:          "Core APU1 temperature. in degrees C.",
		__TEMPERATURE_CURRENT_CHIP2_KEY__:          "Core APU2 temperature. in degrees C.",
		__VOLTAGE_CURRENT_CHIP0_KEY__:              "Current APU0 Voltage. in voltage V.",
		__VOLTAGE_CURRENT_CHIP1_KEY__:              "Current APU1 Voltage. in voltage V.",
		__VOLTAGE_CURRENT_CHIP2_KEY__:              "Current APU2 Voltage. in voltage V.",
		__VOLTAGE_BOARD_INPUT_KEY__:                "Voltage board input.",
		__CLOCKS_CURRENT_APU_CHIP0_KEY__:           "Current apu frequency of APU0 clock.",
		__CLOCKS_CURRENT_APU_CHIP1_KEY__:           "Current apu frequency of APU1 clock.",
		__CLOCKS_CURRENT_APU_CHIP2_KEY__:           "Current apu frequency of APU2 clock.",
		__CLOCKS_CURRENT_CPU_CHIP0_KEY__:           "Current cpu frequency of APU0 clock.",
		__CLOCKS_CURRENT_CPU_CHIP1_KEY__:           "Current cpu frequency of APU1 clock.",
		__CLOCKS_CURRENT_CPU_CHIP2_KEY__:           "Current cpu frequency of APU2 clock.",
		__CLOCKS_CURRENT_MEMORY_CHIP0_KEY__:        "Current memory frequency of APU0 clock.",
		__CLOCKS_CURRENT_MEMORY_CHIP1_KEY__:        "Current memory frequency of APU1 clock.",
		__CLOCKS_CURRENT_MEMORY_CHIP2_KEY__:        "Current memory frequency of APU2 clock.",
		__CLOCKS_MAX_APU_CHIP0_KEY__:               "Current max apu frequency of APU0 clock.",
		__CLOCKS_MAX_APU_CHIP1_KEY__:               "Current max apu frequency of APU1 clock.",
		__CLOCKS_MAX_APU_CHIP2_KEY__:               "Current max apu frequency of APU2 clock.",
		__CLOCKS_MAX_CPU_CHIP0_KEY__:               "Current max cpu frequency of APU0 clock.",
		__CLOCKS_MAX_CPU_CHIP1_KEY__:               "Current max cpu frequency of APU0 clock.",
		__CLOCKS_MAX_CPU_CHIP2_KEY__:               "Current max cpu frequency of APU0 clock.",
		__CLOCKS_MAX_MEMORY_CHIP0_KEY__:            "Current max memory frequency of APU0 clock.",
		__CLOCKS_MAX_MEMORY_CHIP1_KEY__:            "Current max memory frequency of APU1 clock.",
		__CLOCKS_MAX_MEMORY_CHIP2_KEY__:            "Current max memory frequency of APU2 clock.",
		__POWER_DRAW__:                             "The last measured power draw for the entire board, in watts. Only available if power management is supported. This reading is accurate to within +/- 5 watts.",
		__POWER_LIMIT__:                            "The software power limit in watts. Set by software like nvidia-smi. On Kepler devices Power Limit can be adjusted using [-pl | --power-limit=] switches.",
		__ECC_MODE_CURRENT_CHIP0_KEY__:             "Current Ecc mode APU0.",
		__ECC_MODE_CURRENT_CHIP1_KEY__:             "Current Ecc mode APU1.",
		__ECC_MODE_CURRENT_CHIP2_KEY__:             "Current Ecc mode APU2.",
		__ECC_ERRORS_CORRECTED_TOTAL_KEY__:         "Errors detected in global device memory.",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP0_KEY__:   "Errors detected in the APU0",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP1_KEY__:   "Errors detected in the APU1",
		__ECC_ERRORS_CORRECTED_TOTAL_CHIP2_KEY__:   "Errors detected in the APU2",
		__ECC_ERRORS_UNCORRECTED_TOTAL_KEY__:       "Errors detected in global device memory.",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP0_KEY__: "Errors detected in the APU0.",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP1_KEY__: "Errors detected in the APU1.",
		__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP2_KEY__: "Errors detected in the APU2.",
	}
)

const (
	__TIMESTAMP_KEY__                          = "timestamp"
	__BOARD_INDEX_KEY__                        = "board_index"
	__PRODUCT_NAME_KEY__                       = "name"
	__PRODUCT_NUMBER_KEY__                     = "product_number"
	__DRIVER_VERSION_KEY__                     = "driver_version"
	__FIRMWARE_VERSION_KEY__                   = "firmware_version"
	__SERIAL_NUMBER_KEY__                      = "serial_number"
	__CHIP_COUNT_KEY__                         = "chip_count"
	__CHIP_ID_LIST_KEY__                       = "chip_id_list"
	__CHIP_ID_CHIP_KEY__                       = "chip_id.chip"
	__CHIP_ID_CHIP0_KEY__                      = "chip_id.chip0"
	__CHIP_ID_CHIP1_KEY__                      = "chip_id.chip1"
	__CHIP_ID_CHIP2_KEY__                      = "chip_id.chip2"
	__UUID_LIST_KEY__                          = "uuid_list"
	__UUID_CHIP_KEY__                          = "uuid.chip"
	__UUID_CHIP0_KEY__                         = "uuid.chip0"
	__UUID_CHIP1_KEY__                         = "uuid.chip1"
	__UUID_CHIP2_KEY__                         = "uuid.chip2"
	__CHIP_INDEX_LIST_KEY__                    = "chip_index_list"
	__CHIP_INDEX_CHIP_KEY__                    = "chip_index.chip"
	__CHIP_INDEX_CHIP0_KEY__                   = "chip_index.chip0"
	__CHIP_INDEX_CHIP1_KEY__                   = "chip_index.chip1"
	__CHIP_INDEX_CHIP2_KEY__                   = "chip_index.chip2"
	__UTILIZATION_APU_TOTAL_KEY__              = "utilization.apu.total"
	__UTILIZATION_APU_CHIPS_KEY__              = "utilization.apu.chips"
	__UTILIZATION_APU_CHIP_KEY__               = "utilization.apu.chip"
	__UTILIZATION_APU_CHIP0_KEY__              = "utilization.apu.chip0"
	__UTILIZATION_APU_CHIP1_KEY__              = "utilization.apu.chip1"
	__UTILIZATION_APU_CHIP2_KEY__              = "utilization.apu.chip2"
	__UTILIZATION_CPU_TOTAL_KEY__              = "utilization.cpu.total"
	__UTILIZATION_CPU_CHIPS_KEY__              = "utilization.cpu.chips"
	__UTILIZATION_CPU_CHIP_KEY__               = "utilization.cpu.chip"
	__UTILIZATION_CPU_CHIP0_KEY__              = "utilization.cpu.chip0"
	__UTILIZATION_CPU_CHIP1_KEY__              = "utilization.cpu.chip1"
	__UTILIZATION_CPU_CHIP2_KEY__              = "utilization.cpu.chip2"
	__UTILIZATION_VIC_TOTAL_KEY__              = "utilization.vic.total"
	__UTILIZATION_VIC_CHIPS_KEY__              = "utilization.vic.chips"
	__UTILIZATION_VIC_CHIP_KEY__               = "utilization.vic.chip"
	__UTILIZATION_VIC_CHIP0_KEY__              = "utilization.vic.chip0"
	__UTILIZATION_VIC_CHIP1_KEY__              = "utilization.vic.chip1"
	__UTILIZATION_VIC_CHIP2_KEY__              = "utilization.vic.chip2"
	__UTILIZATION_MEMORY_TOTAL_KEY__           = "utilization.memory.total"
	__UTILIZATION_MEMORY_CHIPS_KEY__           = "utilization.memory.chips"
	__UTILIZATION_MEMORY_CHIP_KEY__            = "utilization.memory.chip"
	__UTILIZATION_MEMORY_CHIP0_KEY__           = "utilization.memory.chip0"
	__UTILIZATION_MEMORY_CHIP1_KEY__           = "utilization.memory.chip1"
	__UTILIZATION_MEMORY_CHIP2_KEY__           = "utilization.memory.chip2"
	__UTILIZATION_IPE_FPS_TOTAL_KEY__          = "utilization.ipeFps.total"
	__UTILIZATION_IPE_FPS_CHIPS_KEY__          = "utilization.ipeFps.chips"
	__UTILIZATION_IPE_FPS_CHIP_KEY__           = "utilization.ipeFps.chip"
	__UTILIZATION_IPE_FPS_CHIP0_KEY__          = "utilization.ipeFps.chip0"
	__UTILIZATION_IPE_FPS_CHIP1_KEY__          = "utilization.ipeFps.chip1"
	__UTILIZATION_IPE_FPS_CHIP2_KEY__          = "utilization.ipeFps.chip2"
	__PCI_CHIPS__KEY__                         = "pci.chips"
	__PCI_VENDOR_ID_KEY__                      = "pci.vendor_id"
	__PCI_VENDOR_ID_CHIP_KEY__                 = "pci.vendor_id.chip"
	__PCI_VENDOR_ID_CHIP0_KEY__                = "pci.vendor_id.chip0"
	__PCI_VENDOR_ID_CHIP1_KEY__                = "pci.vendor_id.chip1"
	__PCI_VENDOR_ID_CHIP2_KEY__                = "pci.vendor_id.chip2"
	__PCI_SUB_VENDOR_ID_KEY__                  = "pci.sub_vendor_id"
	__PCI_SUB_VENDOR_ID_CHIP_KEY__             = "pci.sub_vendor_id.chip"
	__PCI_SUB_VENDOR_ID_CHIP0_KEY__            = "pci.sub_vendor_id.chip0"
	__PCI_SUB_VENDOR_ID_CHIP1_KEY__            = "pci.sub_vendor_id.chip1"
	__PCI_SUB_VENDOR_ID_CHIP2_KEY__            = "pci.sub_vendor_id.chip2"
	__PCI_BUS_KEY__                            = "pci.bus"
	__PCI_BUS_CHIP_KEY__                       = "pci.bus.chip"
	__PCI_BUS_CHIP0_KEY__                      = "pci.bus.chip0"
	__PCI_BUS_CHIP1_KEY__                      = "pci.bus.chip1"
	__PCI_BUS_CHIP2_KEY__                      = "pci.bus.chip2"
	__PCI_DEVICE_ID_KEY__                      = "pci.device_id"
	__PCI_DEVICE_ID_CHIP_KEY__                 = "pci.device_id.chip"
	__PCI_DEVICE_ID_CHIP0_KEY__                = "pci.device_id.chip0"
	__PCI_DEVICE_ID_CHIP1_KEY__                = "pci.device_id.chip1"
	__PCI_DEVICE_ID_CHIP2_KEY__                = "pci.device_id.chip2"
	__PCI_SUB_DEVICE_ID_KEY__                  = "pci.sub_device_id"
	__PCI_SUB_DEVICE_ID_CHIP_KEY__             = "pci.sub_device_id.chip"
	__PCI_SUB_DEVICE_ID_CHIP0_KEY__            = "pci.sub_device_id.chip0"
	__PCI_SUB_DEVICE_ID_CHIP1_KEY__            = "pci.sub_device_id.chip1"
	__PCI_SUB_DEVICE_ID_CHIP2_KEY__            = "pci.sub_device_id.chip2"
	__PCI_DEVICE_KEY__                         = "pci.device"
	__PCI_DEVICE_CHIP_KEY__                    = "pci.device.chip"
	__PCI_DEVICE_CHIP0_KEY__                   = "pci.device.chip0"
	__PCI_DEVICE_CHIP1_KEY__                   = "pci.device.chip1"
	__PCI_DEVICE_CHIP2_KEY__                   = "pci.device.chip2"
	__PCI_SUB_FUNCTION_KEY__                   = "pci.function"
	__PCI_SUB_FUNCTION_CHIP_KEY__              = "pci.function.chip"
	__PCI_FUNCTION_CHIP0_KEY__                 = "pci.function.chip0"
	__PCI_FUNCTION_CHIP1_KEY__                 = "pci.function.chip1"
	__PCI_FUNCTION_CHIP2_KEY__                 = "pci.function.chip2"
	__PCI_NUMA_NODE_ID_KEY__                   = "pci.numa.node_id"
	__PCI_NUMA_NODE_ID_CHIP_KEY__              = "pci.numa.node_id.chip"
	__PCI_NUMA_NODE_ID_CHIP0_KEY__             = "pci.numa.node_id.chip0"
	__PCI_NUMA_NODE_ID_CHIP1_KEY__             = "pci.numa.node_id.chip1"
	__PCI_NUMA_NODE_ID_CHIP2_KEY__             = "pci.numa.node_id.chip2"
	__PCI_NUMA_CPU_KEY__                       = "pci.numa.cpu"
	__PCI_NUMA_CPU_CHIP_KEY__                  = "pci.numa.cpu.chip"
	__PCI_NUMA_CPU_CHIP0_KEY__                 = "pci.numa.cpu.chip0"
	__PCI_NUMA_CPU_CHIP1_KEY__                 = "pci.numa.cpu.chip1"
	__PCI_NUMA_CPU_CHIP2_KEY__                 = "pci.numa.cpu.chip2"
	__PCIE_LINK_SPEED_MAX_KEY__                = "pcie.link.speed.max"
	__PCIE_LINK_SPEED_MAX_CHIP_KEY__           = "pcie.link.speed.max.chip"
	__PCIE_LINK_SPEED_MAX_CHIP0_KEY__          = "pcie.link.speed.max.chip0"
	__PCIE_LINK_SPEED_MAX_CHIP1_KEY__          = "pcie.link.speed.max.chip1"
	__PCIE_LINK_SPEED_MAX_CHIP2_KEY__          = "pcie.link.speed.max.chip2"
	__PCIE_LINK_SPEED_CURRENT_KEY__            = "pcie.link.speed.current"
	__PCIE_LINK_SPEED_CURRENT_CHIP_KEY__       = "pcie.link.speed.current.chip"
	__PCIE_LINK_SPEED_CURRENT_CHIP0_KEY__      = "pcie.link.speed.current.chip0"
	__PCIE_LINK_SPEED_CURRENT_CHIP1_KEY__      = "pcie.link.speed.current.chip1"
	__PCIE_LINK_SPEED_CURRENT_CHIP2_KEY__      = "pcie.link.speed.current.chip2"
	__PCIE_LINK_GEN_MAX_KEY__                  = "pcie.link.gen.max"
	__PCIE_LINK_GEN_MAX_CHIP_KEY__             = "pcie.link.gen.max.chip"
	__PCIE_LINK_GEN_MAX_CHIP0_KEY__            = "pcie.link.gen.max.chip0"
	__PCIE_LINK_GEN_MAX_CHIP1_KEY__            = "pcie.link.gen.max.chip1"
	__PCIE_LINK_GEN_MAX_CHIP2_KEY__            = "pcie.link.gen.max.chip2"
	__PCIE_LINK_GEN_CURRENT_KEY__              = "pcie.link.gen.current"
	__PCIE_LINK_GEN_CURRENT_CHIP_KEY__         = "pcie.link.gen.current.chip"
	__PCIE_LINK_GEN_CURRENT_CHIP0_KEY__        = "pcie.link.gen.current.chip0"
	__PCIE_LINK_GEN_CURRENT_CHIP1_KEY__        = "pcie.link.gen.current.chip1"
	__PCIE_LINK_GEN_CURRENT_CHIP2_KEY__        = "pcie.link.gen.current.chip2"
	__FAN_SPEED_KEY__                          = "fan.speed"
	__TEMPERATURE_CURRENT_CHIPS_KEY__          = "temperature.current.chips"
	__TEMPERATURE_CURRENT_CHIP_KEY__           = "temperature.current.chip"
	__TEMPERATURE_CURRENT_CHIP0_KEY__          = "temperature.current.chip0"
	__TEMPERATURE_CURRENT_CHIP1_KEY__          = "temperature.current.chip1"
	__TEMPERATURE_CURRENT_CHIP2_KEY__          = "temperature.current.chip2"
	__VOLTAGE_CURRENT_CHIPS_KEY__              = "voltage.current.chips"
	__VOLTAGE_CURRENT_CHIP_KEY__               = "voltage.current.chip"
	__VOLTAGE_CURRENT_CHIP0_KEY__              = "voltage.current.chip0"
	__VOLTAGE_CURRENT_CHIP1_KEY__              = "voltage.current.chip1"
	__VOLTAGE_CURRENT_CHIP2_KEY__              = "voltage.current.chip2"
	__VOLTAGE_BOARD_INPUT_KEY__                = "voltage.board.input"
	__CLOCKS_KEY__                             = "clocks"
	__CLOCKS_CURRENT_APU_CHIPS_KEY__           = "clocks.current.apu.chips"
	__CLOCKS_CURRENT_APU_KEY__                 = "clocks.current.apu"
	__CLOCKS_CURRENT_APU_CHIP_KEY__            = "clocks.current.apu.chip"
	__CLOCKS_CURRENT_APU_CHIP0_KEY__           = "clocks.current.apu.chip0"
	__CLOCKS_CURRENT_APU_CHIP1_KEY__           = "clocks.current.apu.chip1"
	__CLOCKS_CURRENT_APU_CHIP2_KEY__           = "clocks.current.apu.chip2"
	__CLOCKS_CURRENT_CPU_CHIPS_KEY__           = "clocks.current.cpu.chips"
	__CLOCKS_CURRENT_CPU_KEY__                 = "clocks.current.cpu"
	__CLOCKS_CURRENT_CPU_CHIP_KEY__            = "clocks.current.cpu.chip"
	__CLOCKS_CURRENT_CPU_CHIP0_KEY__           = "clocks.current.cpu.chip0"
	__CLOCKS_CURRENT_CPU_CHIP1_KEY__           = "clocks.current.cpu.chip1"
	__CLOCKS_CURRENT_CPU_CHIP2_KEY__           = "clocks.current.cpu.chip2"
	__CLOCKS_CURRENT_MEMORY_CHIPS_KEY__        = "clocks.current.memory.chips"
	__CLOCKS_CURRENT_MEMORY_KEY__              = "clocks.current.memory"
	__CLOCKS_CURRENT_MEMORY_CHIP_KEY__         = "clocks.current.memory.chip"
	__CLOCKS_CURRENT_MEMORY_CHIP0_KEY__        = "clocks.current.memory.chip0"
	__CLOCKS_CURRENT_MEMORY_CHIP1_KEY__        = "clocks.current.memory.chip1"
	__CLOCKS_CURRENT_MEMORY_CHIP2_KEY__        = "clocks.current.memory.chip2"
	__CLOCKS_MAX_APU_CHIPS_KEY__               = "clocks.current.apu.max.chips"
	__CLOCKS_MAX_APU_CHIP_KEY__                = "clocks.current.apu.max.chip"
	__CLOCKS_MAX_APU_CHIP0_KEY__               = "clocks.current.apu.max.chip0"
	__CLOCKS_MAX_APU_CHIP1_KEY__               = "clocks.current.apu.max.chip1"
	__CLOCKS_MAX_APU_CHIP2_KEY__               = "clocks.current.apu.max.chip2"
	__CLOCKS_MAX_CPU_CHIPS_KEY__               = "clocks.current.cpu.max.chips"
	__CLOCKS_MAX_CPU_CHIP_KEY__                = "clocks.current.cpu.max.chip"
	__CLOCKS_MAX_CPU_CHIP0_KEY__               = "clocks.current.cpu.max.chip0"
	__CLOCKS_MAX_CPU_CHIP1_KEY__               = "clocks.current.cpu.max.chip1"
	__CLOCKS_MAX_CPU_CHIP2_KEY__               = "clocks.current.cpu.max.chip2"
	__CLOCKS_MAX_MEMORY_CHIPS_KEY__            = "clocks.current.memory.max.chips"
	__CLOCKS_MAX_MEMORY_CHIP_KEY__             = "clocks.current.memory.max.chip"
	__CLOCKS_MAX_MEMORY_CHIP0_KEY__            = "clocks.current.memory.max.chip0"
	__CLOCKS_MAX_MEMORY_CHIP1_KEY__            = "clocks.current.memory.max.chip1"
	__CLOCKS_MAX_MEMORY_CHIP2_KEY__            = "clocks.current.memory.max.chip2"
	__POWER_DRAW__                             = "power.draw"
	__POWER_LIMIT__                            = "power.limit"
	__ECC_MODE_CURRENT_CHIPS_KEY__             = "ecc.mode.current.chips"
	__ECC_MODE_CURRENT_CHIP_KEY__              = "ecc.mode.current.chip"
	__ECC_MODE_CURRENT_CHIP0_KEY__             = "ecc.mode.current.chip0"
	__ECC_MODE_CURRENT_CHIP1_KEY__             = "ecc.mode.current.chip1"
	__ECC_MODE_CURRENT_CHIP2_KEY__             = "ecc.mode.current.chip2"
	__ECC_ERRORS_CORRECTED_TOTAL_KEY__         = "ecc.errors.corrected.total"
	__ECC_ERRORS_CORRECTED_TOTAL_CHIPS_KEY__   = "ecc.errors.corrected.total.chips"
	__ECC_ERRORS_CORRECTED_TOTAL_CHIP_KEY__    = "ecc.errors.corrected.total.chip"
	__ECC_ERRORS_CORRECTED_TOTAL_CHIP0_KEY__   = "ecc.errors.corrected.total.chip0"
	__ECC_ERRORS_CORRECTED_TOTAL_CHIP1_KEY__   = "ecc.errors.corrected.total.chip1"
	__ECC_ERRORS_CORRECTED_TOTAL_CHIP2_KEY__   = "ecc.errors.corrected.total.chip2"
	__ECC_ERRORS_UNCORRECTED_TOTAL_KEY__       = "ecc.errors.uncorrected.total"
	__ECC_ERRORS_UNCORRECTED_TOTAL_CHIPS_KEY__ = "ecc.errors.uncorrected.total.chips"
	__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP_KEY__  = "ecc.errors.uncorrected.total.chip"
	__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP0_KEY__ = "ecc.errors.uncorrected.total.chip0"
	__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP1_KEY__ = "ecc.errors.uncorrected.total.chip1"
	__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP2_KEY__ = "ecc.errors.uncorrected.total.chip2"
)

type BoardBaseInfo struct {
	TimeStamp                     string               `json:"timestamp"`
	BoardIndex                    string               `json:"board_index"`
	ProductName                   string               `json:"name"`
	ProductBrand                  string               `json:"product_brand"`
	ProductNumber                 string               `json:"product_number"`
	DriverVersion                 string               `json:"driver_version"`
	FirmwareVersion               string               `json:"firmware_version"`
	SerialNumber                  string               `json:"serial_number"`
	UuidList                      []string             `json:"uuid_list"`
	ChipCount                     string               `json:"chip_count"`
	ChipIdList                    []string             `json:"chip_id_list"`
	ChipIndexList                 []string             `json:"chip_index_list"`
	ApuTotal                      string               `json:"utilization.apu.total"`
	ApuUtilList                   []string             `json:"utilization.apu.chips"`
	CpuTotal                      string               `json:"utilization.cpu.total"`
	CpuUtilList                   []string             `json:"utilization.cpu.chips"`
	VicTotal                      string               `json:"utilization.vic.total"`
	VicUtilList                   []string             `json:"utilization.vic.chips"`
	MemoryTotal                   string               `json:"utilization.memory.total"`
	MemoryUtilList                []string             `json:"utilization.memory.chips"`
	IpeTotal                      string               `json:"utilization.ipeFps.total"`
	IpeUtilList                   []string             `json:"utilization.ipeFps.chips"`
	TempUtilList                  []string             `json:"temperature.current.chips"`
	FanSpeed                      string               `json:"fan.speed"`
	ChipVoltageList               []string             `json:"voltage.current.chips"`
	VoltageInput                  string               `json:"voltage.board.input"`
	PowerDraw                     string               `json:"power.draw"`
	PowerLimit                    string               `json:"power.limit"`
	EccModeList                   []string             `json:"ecc.mode.current.chips"`
	DdrEccCorrectedTotal          string               `json:"ecc.errors.corrected.total"`
	DdrEccUnCorrectedTotal        string               `json:"ecc.errors.uncorrected.total"`
	DdrEccErrCorrectedChipCount   []string             `json:"ecc.errors.corrected.total.chips"`
	DdrEccErrUnCorrectedChipCount []string             `json:"ecc.errors.uncorrected.total.chips"`
	PciInfoList                   []BoardPciDeviceInfo `json:"pci.chips"`
	ClocksInfo                    BoardClocksInfo      `json:"clocks"`
}

type BoardClocksInfo struct {
	ApuClocksList       []string `json:"clocks.current.apu.chips"`
	CpuClocksList       []string `json:"clocks.current.cpu.chips"`
	MemoryClocksList    []string `json:"clocks.current.memory.chips"`
	ApuClocksMaxList    []string `json:"clocks.current.apu.max.chips"`
	CpuClocksMaxList    []string `json:"clocks.current.cpu.max.chips"`
	MemoryClocksMaxList []string `json:"clocks.current.memory.max.chips"`
}

type BoardPciDeviceInfo struct {
	VendorId     string `json:"pci.vendor_id"`
	DeviceId     string `json:"pci.device_id"`
	SubVendorId  string `json:"pci.sub_vendor_id"`
	SubDeviceId  string `json:"pci.sub_device_id"`
	BusNum       string `json:"pci.bus"`
	Device       string `json:"pci.device"`
	Function     string `json:"pci.function"`
	MaxSpeed     string `json:"pcie.link.speed.max"`
	MaxWidth     string `json:"pcie.link.gen.max"`
	CurrentSpeed string `json:"pcie.link.speed.current"`
	CurrentWidth string `json:"pcie.link.gen.current"`
	NumaNodeId   string `json:"pci.numa.node_id"`
	NumaCPUList  string `json:"pci.numa.cpu"`
}

func (pdi *BoardPciDeviceInfo) Set(vendorId string, deviceId string, subVendorId string,
	subDeviceId string, busNum string, device string, function string, maxSpeed string,
	maxWidth string, currentSpeed string, currentWidth string, numaNodeId string, numCPUList string) {
	pdi.VendorId = vendorId
	pdi.DeviceId = deviceId
	pdi.SubVendorId = subVendorId
	pdi.SubDeviceId = subDeviceId
	pdi.BusNum = busNum
	pdi.Device = device
	pdi.Function = function
	pdi.MaxSpeed = maxSpeed
	pdi.MaxWidth = maxWidth
	pdi.CurrentSpeed = currentSpeed
	pdi.CurrentWidth = currentWidth
	pdi.NumaNodeId = numaNodeId
	pdi.NumaCPUList = numCPUList
}

const (
	DefaultPciDevicesInfoPath = "/sys/bus/pci/devices/0000:" // /sys/bus/pci/devices/0000:e5:00.0
	NumaNode                  = "numa_node"
	NumaNodeCPUList           = "local_cpulist"
	CurrentLinkSpeed          = "current_link_speed"
	CurrentLinkWidth          = "current_link_width"
	MaxLinkWidth              = "max_link_width"
	MaxLinkSpeed              = "max_link_speed"
	DeviceID                  = "device"
	VendorID                  = "vendor"
	SubSystemDevice           = "subsystem_device"
	SubSystemVendor           = "subsystem_vendor"
)

const (
	__BOARD_STR__             = "Board:"
	__PRODUCT_NAME_STR__      = "Product Name"
	__PRODUCT_BRAND_STR__     = "Product Brand"
	__PRODUCT_NUMBER_STR__    = "Product Number"
	__DRIVER_STR__            = "Driver Version"
	__FIRMWARE_VERSION_STR__  = "Firmware Version"
	__SERIAL_NUMBER_STR__     = "Serial Number"
	__CHIP_COUNT_STR__        = "Chip Count"
	__CHIP_INDEX_STR__        = "Chip Index"
	__CHIP_ID_STR__           = "Chip ID"
	__ECID_STR__              = "ECID"
	__UUID_STR__              = "UUID"
	__UTILIZATION_STR__       = "Utilization"
	__APU_STR__               = "APU"
	__TOTAL_STR__             = "Total"
	__CPU_STR__               = "CPU"
	__VIC_STR__               = "VIC"
	__MEMORY_STR__            = "Memory"
	__IPE_FPS_STR__           = "IPE-FPS"
	__IPE_STR__               = "IPE"
	__PCIE_STR__              = "PCIE"
	__PCI_STR__               = "PCI"
	__VENDOR_ID_STR__         = "Vendor ID"
	__DEVICE_ID_STR__         = "Device ID"
	__SUB_DEVICE_ID_STR__     = "Sub Device ID"
	__SUB_Vendor_ID_STR__     = "Sub Vendor ID"
	__PCIE_GEN_STR__          = "PCIe Generation"
	__GENERATION_STR__        = "Generation"
	__MAX_STR__               = "Max"
	__CURRENT__               = "Current"
	__BUS_NUM__               = "Bus num"
	__DEVICE__                = "Device"
	__FUNCTION__              = "Function"
	__PHYSICAL_SLOT__         = "Physical Slot"
	__NUMA_NODE_ID__          = "NUMA node id"
	__FAN_STR__               = "Fan"
	__TEMPERATURE_STR__       = "Temperature"
	__BIU_CURRENT_TEMP_STR__  = "BIU Current Temp"
	__BIU_SLOWDOWN_TEMP_STR__ = "BIU Slowdown Temp"
	__BIU_SHUTDOWN_TEMP_STR__ = "BIU Shutdown Temp"
	__VOLTAGE_STR__           = "Voltage"
	__CHIP_VOLTAGE__STR__     = "Chip Voltage"
	__BOARD_VOLTAGE_STR__     = "Board Voltage"
	__CLOCKS_STR__            = "Clocks"
	__INPUT_STR__             = "Input"
	__PCIE_SWITCH__           = "PCIe Switch"
	__APU_CLOCK_STR__         = "APU Clock"
	__APU_MAX_CLOCK_STR__     = "APU Max Clock"
	__CPU_CLOCK_STR__         = "CPU Clock"
	__CPU_MAX_CLOCK_STR__     = "CPU Max Clock"
	__MEMORY_CLOCK_STR__      = "Memory Clock"
	__MEMORY_MAX_CLOCK_STR__  = "Memory Max Clock"
	__POWER_STR__             = "Power"
	__POWER_DRAW_STR__        = "Power Draw"
	__POWER_LIMIT_STR__       = "Power Limit"
	__ECC_MODE_STR__          = "ECC Mode"
	__DDR_ECC_ERR_COUNT_STR__ = "DDR ECC Err Count"
	__CORRECTED_ERR_STR__     = "Corrected Err"
	__UNCORRECTED_ERR_STR__   = "Uncorrected Err"
	__CHIP0_STR__             = "Chip0"
	__CHIP1_STR__             = "Chip1"
	__CHIP2_STR__             = "Chip2"
	__CHIP0_COLON_STR__       = "Chip0"
	__CHIP1_COLON_STR__       = "Chip1"
	__CHIP2_COLON_STR__       = "Chip2"
	__CHIP_STR__              = "Chip"
)

const (
	__SN_STR__      = "SN"
	__START_STR__   = "***"
	__ERROR_STR__   = "ERROR"
	__SUB_STR__     = "Sub"
	__NA_STR__      = "NA"
	__N_A_STR__     = "N/A"
	__UNKNOWN_STR__ = "unknown"
	__APU_SMI_STR__ = "APU-SMI"
)

var (
	VersionCmdParam                    []string = []string{"-l", "|", "grep", "-i"}
	LynChipDeviceIDAndVendorIDCmdParam []string = strings.Split("-d 1e9f:27c5", __SPCAE_SEP__)
	LynSmiDetailInfoCmdParam           string   = "-q"
	LynSmiCardIdCmdParam               string   = "-i"
	LynSmiChipIdCmdParam               string   = "-c"
	LynSmiVersionCmdParam              string   = "-v"
)

func toQFieldSlice(ss []string) []qField {
	r := make([]qField, len(ss))
	for i, s := range ss {
		r[i] = qField(s)
	}
	return r
}

func getKeys(m map[qField]rField) []qField {
	r := make([]qField, len(m))
	i := 0
	for key := range m {
		r[i] = key
		i++
	}
	return r
}

func parserVersionStr(v string) string {
	re := regexp.MustCompile(`^ii*\s+\w+\s+(\d+\.\d+\.\d+?)\s+\w*`)
	match := re.FindStringSubmatch(v)
	return VersionShortStr + string(match[1])
}

func isStrBlank(str string) bool {
	return str == __SPCAE_SEP__
}

func removeDebugInfo(line *string, r *bufio.Reader, err error) {
	for err == nil {
		if strings.Contains(*line, "ERROR") || strings.Contains(*line, "lynSmi.cpp") || strings.Contains(*line, __SN_STR__) || strings.Contains(*line, __START_STR__) {
			log.Debugln(*line)
			*line, err = r.ReadString(__LINE_FEED_SEP__)
		} else {
			break
		}
		*line, err = r.ReadString(__LINE_FEED_SEP__)
	}
}

func getVersion(v Version) string {
	switch v {
	case SDK:
		var SDKVersion string
		fn := func(info string) {
			SDKVersion = info
		}
		RunShellCmdGetVersionInfo(fn, DefaultFindLynDriverCommand, append(VersionCmdParam, LynSdkStr)...)
		return SDKVersion
	case Driver:
		var DriverVersion string
		fn := func(info string) {
			if strings.Contains(info, LynDriverStr) {
				DriverVersion = info
			}
		}
		RunShellCmdGetVersionInfo(fn, DefaultFindLynDriverCommand, append(VersionCmdParam, LynDriverStr)...)
		DriverVersion = parserVersionStr(DriverVersion)
		return DriverVersion
	case SMI:
		var smiVersion string
		fn := func(info string) {
			smiVersion = strings.Split(info, __COLON_SEP__)[1]
		}
		RunShellCmdGetVersionInfo(fn, DefaultLynSmiCommand, LynSmiVersionCmdParam)
		return VersionShortStr + strings.Replace(strings.TrimSpace(smiVersion), string(__LINE_FEED_SEP__), "", -1)
	}

	return __UNKNOWN_STR__
}

func replaceNAInfo(line string) string {
	if strings.Contains(line, __NA_STR__) {
		return strings.Replace(line, __NA_STR__, __N_A_STR__, -1)
	} else {
		return line
	}
}

func replaceDriverVersionInfo(line string) string {
	if strings.Contains(line, __DRIVER_STR__) {
		driverInfoSlice := strings.Split(line, __SPCAE_SEP__)
		driverInfoSlice[len(driverInfoSlice)-1] = getVersion(Driver) + "\n"
		return strings.Join(driverInfoSlice, __SPCAE_SEP__)
	} else {
		return line
	}
}

func replaceProductNameToSmiVersionName(line string) string {
	if strings.Contains(line, __PRODUCT_NAME_STR__) {
		line = strings.Replace(line, __PRODUCT_NAME_STR__, __APU_SMI_STR__, -1)
		strSlice := strings.Split(line, __SPCAE_SEP__)
		return strings.Replace(line, string(strSlice[1]), getVersion(SMI), -1)
	} else {
		return line
	}
}

func replaceECIDInfoToChipIndex(line string, chipCount int, r *bufio.Reader, chipIndex int) string {
	if strings.Contains(line, __ECID_STR__) {
		line = strings.Replace(line, __ECID_STR__, __CHIP_INDEX_STR__, -1)
		fmt.Print(line)
		for i := 0; i < chipCount; i++ {
			line, _ = r.ReadString(__LINE_FEED_SEP__)
			uuidInfoSlice := strings.Split(line, __SPCAE_SEP__)
			uuidInfoSlice[len(uuidInfoSlice)-1] = strconv.Itoa(chipIndex) + "\n"
			line = strings.Join(uuidInfoSlice, __SPCAE_SEP__)
			chipIndex++
			fmt.Print(line)
		}
		line, _ = r.ReadString(__LINE_FEED_SEP__)
		return line
	} else {
		return line
	}
}

func replacePciInfo(line string, deviceInfo string) string {
	str_slice := strings.Split(line, __SPCAE_SEP__)
	str_slice[len(str_slice)-1] = deviceInfo
	return strings.Join(str_slice, __SPCAE_SEP__)
}

func updatePciInfo(line string, pciChipIndex int, chipIndex int, pciInfoStrList []string) string {
	switch {
	case strings.Contains(line, __PCIE_STR__) && !strings.Contains(line, __GENERATION_STR__):
		return strings.Replace(line, __PCIE_STR__, __PCI_STR__, -1)
	case strings.Contains(line, __VENDOR_ID_STR__) && !strings.Contains(line, __SUB_STR__):
		vendorId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, VendorId)
		return replacePciInfo(line, vendorId)
	case strings.Contains(line, __DEVICE_ID_STR__) && !strings.Contains(line, __SUB_STR__):
		deviceId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, DeviceId)
		return replacePciInfo(line, deviceId)
	case strings.Contains(line, __SUB_Vendor_ID_STR__) && strings.Contains(line, __SUB_STR__):
		subVendorId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, SubVendorId)
		return replacePciInfo(line, subVendorId)
	case strings.Contains(line, __SUB_DEVICE_ID_STR__) && strings.Contains(line, __SUB_STR__):
		subDeviceId, pciDeviceInfo := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, SubDeviceId)
		maxSpeed, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, MaxSpeed)
		currentSpeed, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, CurrentSpeed)
		numaNodeId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, NumaNodeId)
		numaCpuList, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, NumaCPUList)
		pciInfoSlice := strings.Split(pciDeviceInfo, __COLON_SEP__)
		devAndFunc := strings.Split(pciInfoSlice[1], __DOT_SEP__)
		str_slice := strings.Split(line, __SPCAE_SEP__)[0:8]
		space := strings.Join(str_slice, __SPCAE_SEP__)
		pciInfo := space + " Bus Num:" + space + space + pciInfoSlice[0] + "\n" + space + " Device: " + space + space + devAndFunc[0] + "\n" + space + " Function:      " + space + devAndFunc[1] + "\n"
		pcieInfo := space + " Max Speed:     " + space + maxSpeed + space + " Current Speed: " + space + currentSpeed + space + " NumaNodeId:    " + space + numaNodeId + space + " NumaCpuList:   " + space + numaCpuList
		return replacePciInfo(line, subDeviceId) + pciInfo + pcieInfo
	}
	return line
}

func queryPciDeviceInfoByDeviceID(devicePciInfo string, info PciInfo) string {
	const sep = "/"
	switch info {
	case VendorId:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + VendorID)
	case DeviceId:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + DeviceID)
	case SubVendorId:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + SubSystemVendor)
	case SubDeviceId:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + SubSystemDevice)
	case BusNum:
		return strings.Split(devicePciInfo, __COLON_SEP__)[0]
	case Device:
		pciInfoSlice := strings.Split(devicePciInfo, __COLON_SEP__)
		return strings.Split(pciInfoSlice[1], __DOT_SEP__)[0]
	case Function:
		pciInfoSlice := strings.Split(devicePciInfo, __COLON_SEP__)
		return strings.Split(pciInfoSlice[1], __DOT_SEP__)[1]
	case MaxSpeed:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + MaxLinkSpeed)
	case MaxWidth:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + MaxLinkWidth)
	case CurrentSpeed:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + CurrentLinkSpeed)
	case CurrentWidth:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + CurrentLinkWidth)
	case NumaNodeId:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + NumaNode)
	case NumaCPUList:
		return ReadPciInfoStr(DefaultPciDevicesInfoPath + devicePciInfo + sep + NumaNodeCPUList)
	}
	return fmt.Sprintf("%s get failed", devicePciInfo)
}

func getBoardIndex(line string) (string, int) {
	if strings.Contains(line, __BOARD_STR__) {
		boardIndexSlice := strings.Split(line, __COLON_SEP__)
		return strings.TrimSpace(boardIndexSlice[len(boardIndexSlice)-1]), 1
	}
	return line, -1
}

func computeCurrentChipIndex(boardIndex int, chipCount int) int {
	return (boardIndex+1)*chipCount - chipCount
}

func getPciDeviceInfo(pciChipIndex int, chipIndex int, pciInfoStrList []string, info PciInfo) (string, string) {
	switch pciChipIndex {
	case 0:
		PciDeviceInfo := pciInfoStrList[chipIndex]
		return queryPciDeviceInfoByDeviceID(PciDeviceInfo, info), PciDeviceInfo
	case 1:
		PciDeviceInfo := pciInfoStrList[chipIndex+1]
		return queryPciDeviceInfoByDeviceID(PciDeviceInfo, info), PciDeviceInfo
	case 2:
		PciDeviceInfo := pciInfoStrList[chipIndex+2]
		return queryPciDeviceInfoByDeviceID(PciDeviceInfo, info), PciDeviceInfo
	default:
		PciDeviceInfo := pciInfoStrList[:]
		log.Debugln(PciDeviceInfo)
		return "get info failed", "get info failed"
	}
}

func removeLineBreak(info string) string {
	return strings.Replace(info, __LINE_FEED_STR__, "", -1)
}

func computePciChipIndex(line string, currentBoardIndex *int, boardIndex *int, pciChipIndex *int) {
	if strings.Contains(line, __VENDOR_ID_STR__) && !strings.Contains(line, __SUB_STR__) {
		if *currentBoardIndex == *boardIndex {
			*pciChipIndex++
		} else {
			*currentBoardIndex = *boardIndex
			*pciChipIndex = 0
		}
	}
}

func getClocksVal(r *bufio.Reader, infoList []string, index int) {
	line, _ := r.ReadString(__LINE_FEED_SEP__)
	infoList[index] = strings.Split(getBoardInfoVal(line), __SPCAE_SEP__)[0]
}

func getBoardUtilTotalVal(utilStr string) string {
	indexVal := 1
	if strings.Contains(utilStr, __PER_SEP__) {
		indexVal = 2
	}
	utilStrSlice := strings.Split(utilStr, __SPCAE_SEP__)
	return strings.TrimSpace(utilStrSlice[len(utilStrSlice)-indexVal])
}

func getBoardUtilInfo(line string, chipCount string, r *bufio.Reader) []string {
	count, _ := strconv.Atoi(chipCount)
	var utilStrList []string
	for i := 0; i < count; i++ {
		line, _ = r.ReadString(__LINE_FEED_SEP__)
		boardUtilInfo := getBoardUtilTotalVal(line)
		utilStrList = append(utilStrList, boardUtilInfo)
	}
	return utilStrList
}

func getChipIndexByChipCountAndBoardIndex(chipStartIndex int, chipCount string) []string {
	count, _ := strconv.Atoi(chipCount)
	var chipIndexList []string
	for i := chipStartIndex; i < chipStartIndex+count; i++ {
		chipIndex := strconv.Itoa(i)
		chipIndexList = append(chipIndexList, chipIndex)
	}
	return chipIndexList
}

func getChipCount(r *bufio.Reader, line *string, fn func(line *string)) int {
	var chipIdList []string
	fn(line)
	for {
		*line, _ = r.ReadString(__LINE_FEED_SEP__)
		if strings.Contains(*line, __CHIP_STR__) {
			fn(line)
			chipIdList = append(chipIdList, getBoardInfoVal(*line))
		} else {
			break
		}
	}
	return len(chipIdList)
}

func getChipCountByChipId(r *bufio.Reader, line *string) int {
	fn := func(line *string) {
	}
	return getChipCount(r, line, fn)
}

func getChipCountByChipIdAndPrint(r *bufio.Reader, line *string) int {
	fn := func(line *string) {
		fmt.Print(*line)
	}
	return getChipCount(r, line, fn)
}

func getChipIdListByChipCount(r *bufio.Reader, line *string) ([]string, string) {
	var chipIdList []string
	for {
		*line, _ = r.ReadString(__LINE_FEED_SEP__)
		if strings.Contains(*line, __CHIP_STR__) {
			chipIdList = append(chipIdList, getBoardInfoVal(*line))
		} else {
			break
		}
	}
	_chipCount := strconv.Itoa(len(chipIdList))
	return chipIdList, _chipCount
}

func getBoardInfoVal(infoStr string) string {
	var (
		infoStrSlice []string
		val          string
	)
	if strings.Contains(infoStr, __COLON_SEP__) {
		infoStrSlice = strings.Split(infoStr, __COLON_SEP__)
		val = strings.TrimSpace(infoStrSlice[len(infoStrSlice)-1])
	} else {
		infoStrSlice = strings.Split(infoStr, __SPCAE_SEP__)
		_index := len(infoStrSlice) - 2
		if strings.Contains(infoStr, __UNCORRECTED_ERR_STR__) && !strings.Contains(infoStr, __COLON_SEP__) {
			_index = len(infoStrSlice) - 1
		}
		val = strings.TrimSpace(infoStrSlice[_index])
	}
	if strings.Contains(val, __V_SEP__) || strings.Contains(val, __C_SEP__) || strings.Contains(val, __W_SEP__) {
		return strings.Split(val, __SPCAE_SEP__)[0]
	}
	return val
}

func removeDuplicateQFields(qFields []qField) []qField {
	m := make(map[qField]struct{})
	var r []qField
	for _, f := range qFields {
		_, exists := m[f]
		if !exists {
			r = append(r, f)
			m[f] = struct{}{}
		}
	}
	return r
}

func getBoardSN(line string) (string, int) {
	if strings.Contains(line, __SERIAL_NUMBER_STR__) {
		boardSnSlice := strings.Split(line, __COLON_SEP__)
		return strings.TrimSpace(boardSnSlice[len(boardSnSlice)-1]), 1
	}
	return line, -1
}

func getBoardPN(line string) (string, int) {
	if strings.Contains(line, __PRODUCT_NAME_STR__) {
		boardSnSlice := strings.Split(line, __COLON_SEP__)
		return strings.TrimSpace(boardSnSlice[len(boardSnSlice)-1]), 1
	}
	return line, -1
}

func removeAndReplaceBaseInfo(line string) string {
	line = replaceNAInfo(line)
	line = replaceDriverVersionInfo(line)
	return line
}

func printAPUsInfoTitle(q []qField) {
	for i, v := range q {
		if len(q)-1 != i {
			fmt.Print(v + __COMMA_SEP__ + __SPCAE_SEP__)
		} else {
			fmt.Println(v)
		}
	}
}

func verifyAndCheckQueryFields(qFieldsRaw string) ([]qField, error) {
	qFieldsSeparated := strings.Split(qFieldsRaw, __COMMA_SEP__)
	qFields := toQFieldSlice(qFieldsSeparated)
	qFields = removeDuplicateQFields(qFields)
	fieldMap := fallbackQFieldToRFieldMap
	for _, f := range qFields {
		_, exists := fieldMap[f]
		if !exists {
			fmt.Printf("Field %s is not a valid field to query.\n", strconv.Quote(string(f)))
			return qFields, errors.New(string("field " + f + " verify Failed"))
		}
	}
	return qFields, nil
}

func getAPUInfoByBoardMapInfo(mapData map[string]string, q qField) rField {
	return rField(mapData[string(q)])
}

func flatMapDataToFlatDataStringMapData(flatMapData map[string]interface{}) map[string]string {
	m := make(map[string]string, len(flatMapData))
	for k, v := range flatMapData {
		switch v.(type) {
		case string:
			m[k] = v.(string)
		}
	}
	return m
}

func getLynSmiDetailInfo(r *bufio.Reader, cmd *exec.Cmd) {
	var chipCount int = 0
	var boardIndex int = 0
	var pciChipIndex int = -1
	var currentBoardIndex int = 0
	ch := make(chan []string)
	line, err := r.ReadString(__LINE_FEED_SEP__)
	go QueryLynPciInfo(ch)
	pciInfoStrList := <-ch
	removeDebugInfo(&line, r, err)
	for {
		if err != nil || io.EOF == err {
			break
		}
		line = removeAndReplaceBaseInfo(line)
		boardIndexStr, errCode := getBoardIndex(line)
		if errCode != -1 {
			boardIndex, _ = strconv.Atoi(boardIndexStr)
		}
		if !isStrBlank(line) {
			line = replaceDriverVersionInfo(line)
			if strings.Contains(line, __CHIP_ID_STR__) {
				chipCount = getChipCountByChipIdAndPrint(r, &line)
			}

			chipIndex := computeCurrentChipIndex(boardIndex, chipCount)
			line = replaceECIDInfoToChipIndex(line, chipCount, r, chipIndex)
			computePciChipIndex(line, &currentBoardIndex, &boardIndex, &pciChipIndex)
			line = updatePciInfo(line, pciChipIndex, chipIndex, pciInfoStrList)
			fmt.Print(line)
		}
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	err = cmd.Wait()
	if err != nil {
		log.Debugln(err)
	}
}

func apusInfoToFlatMap(mapData map[string]interface{}) map[string]string {
	chip_count, _ := strconv.Atoi(mapData[__CHIP_COUNT_KEY__].(string))
	uuid_list := getMapDataSliceValByKeyName(__UUID_LIST_KEY__, mapData)
	chip_id_list := getMapDataSliceValByKeyName(__CHIP_ID_LIST_KEY__, mapData)
	chip_index_list := getMapDataSliceValByKeyName(__CHIP_INDEX_LIST_KEY__, mapData)
	util_apu_chips := getMapDataSliceValByKeyName(__UTILIZATION_APU_CHIPS_KEY__, mapData)
	util_cpu_chips := getMapDataSliceValByKeyName(__UTILIZATION_CPU_CHIPS_KEY__, mapData)
	util_memory_chips := getMapDataSliceValByKeyName(__UTILIZATION_MEMORY_CHIPS_KEY__, mapData)
	util_vic_chips := getMapDataSliceValByKeyName(__UTILIZATION_VIC_CHIPS_KEY__, mapData)
	util_ipe_chips := getMapDataSliceValByKeyName(__UTILIZATION_IPE_FPS_CHIPS_KEY__, mapData)
	temp_current_chips := getMapDataSliceValByKeyName(__TEMPERATURE_CURRENT_CHIPS_KEY__, mapData)
	voltage_current_chips := getMapDataSliceValByKeyName(__VOLTAGE_CURRENT_CHIPS_KEY__, mapData)
	ecc_mode_current_chips := getMapDataSliceValByKeyName(__ECC_MODE_CURRENT_CHIPS_KEY__, mapData)
	ecc_errors_corrected_chips := getMapDataSliceValByKeyName(__ECC_ERRORS_CORRECTED_TOTAL_CHIPS_KEY__, mapData)
	ecc_errors_uncorrected_chips := getMapDataSliceValByKeyName(__ECC_ERRORS_UNCORRECTED_TOTAL_CHIPS_KEY__, mapData)
	pci_chips := getMapDataSliceValByKeyName(__PCI_CHIPS__KEY__, mapData)
	clocks := getMapDataMapValByKeyName(__CLOCKS_KEY__, mapData)

	for i := 0; i < chip_count; i++ {
		keyName := getMapDataKeyIndex(__UUID_CHIP_KEY__, i)
		mapData[keyName] = uuid_list[i]
		keyName = getMapDataKeyIndex(__CHIP_ID_CHIP_KEY__, i)
		mapData[keyName] = chip_id_list[i]
		keyName = getMapDataKeyIndex(__CHIP_INDEX_CHIP_KEY__, i)
		mapData[keyName] = chip_index_list[i]
		keyName = getMapDataKeyIndex(__UTILIZATION_APU_CHIP_KEY__, i)
		mapData[keyName] = util_apu_chips[i]
		keyName = getMapDataKeyIndex(__UTILIZATION_CPU_CHIP_KEY__, i)
		mapData[keyName] = util_cpu_chips[i]
		keyName = getMapDataKeyIndex(__UTILIZATION_MEMORY_CHIP_KEY__, i)
		mapData[keyName] = util_memory_chips[i]
		keyName = getMapDataKeyIndex(__UTILIZATION_VIC_CHIP_KEY__, i)
		mapData[keyName] = util_vic_chips[i]
		keyName = getMapDataKeyIndex(__UTILIZATION_IPE_FPS_CHIP_KEY__, i)
		mapData[keyName] = util_ipe_chips[i]
		keyName = getMapDataKeyIndex(__TEMPERATURE_CURRENT_CHIP_KEY__, i)
		mapData[keyName] = temp_current_chips[i]
		keyName = getMapDataKeyIndex(__VOLTAGE_CURRENT_CHIP_KEY__, i)
		mapData[keyName] = voltage_current_chips[i]
		keyName = getMapDataKeyIndex(__ECC_MODE_CURRENT_CHIP_KEY__, i)
		mapData[keyName] = ecc_mode_current_chips[i]
		keyName = getMapDataKeyIndex(__ECC_ERRORS_CORRECTED_TOTAL_CHIP_KEY__, i)
		mapData[keyName] = ecc_errors_corrected_chips[i]
		keyName = getMapDataKeyIndex(__ECC_ERRORS_UNCORRECTED_TOTAL_CHIP_KEY__, i)
		mapData[keyName] = ecc_errors_uncorrected_chips[i]
		keyName = getMapDataKeyIndex(__PCI_VENDOR_ID_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_VENDOR_ID_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_DEVICE_ID_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_DEVICE_ID_CHIP_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_SUB_VENDOR_ID_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_SUB_VENDOR_ID_CHIP_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_SUB_DEVICE_ID_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_SUB_DEVICE_ID_CHIP_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_BUS_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_BUS_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_DEVICE_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_DEVICE_ID_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_SUB_FUNCTION_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_SUB_FUNCTION_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCIE_LINK_SPEED_CURRENT_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCIE_LINK_SPEED_CURRENT_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCIE_LINK_GEN_CURRENT_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCIE_LINK_GEN_CURRENT_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCIE_LINK_SPEED_MAX_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCIE_LINK_SPEED_MAX_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCIE_LINK_GEN_MAX_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCIE_LINK_GEN_MAX_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_NUMA_NODE_ID_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_NUMA_NODE_ID_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__PCI_NUMA_CPU_CHIP_KEY__, i)
		mapData[keyName] = getMapDataDeepMapValByKeyName(__PCI_NUMA_CPU_KEY__, i, pci_chips)
		keyName = getMapDataKeyIndex(__CLOCKS_CURRENT_APU_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_CURRENT_APU_CHIPS_KEY__, i, clocks)
		keyName = getMapDataKeyIndex(__CLOCKS_CURRENT_CPU_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_CURRENT_CPU_CHIPS_KEY__, i, clocks)
		keyName = getMapDataKeyIndex(__CLOCKS_CURRENT_MEMORY_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_CURRENT_MEMORY_CHIPS_KEY__, i, clocks)
		keyName = getMapDataKeyIndex(__CLOCKS_MAX_APU_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_MAX_APU_CHIPS_KEY__, i, clocks)
		keyName = getMapDataKeyIndex(__CLOCKS_MAX_CPU_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_MAX_CPU_CHIPS_KEY__, i, clocks)
		keyName = getMapDataKeyIndex(__CLOCKS_MAX_MEMORY_CHIP_KEY__, i)
		mapData[keyName] = getMapDataMapSliceValByKeyName(__CLOCKS_MAX_MEMORY_CHIPS_KEY__, i, clocks)
	}
	delete(mapData, __UUID_LIST_KEY__)
	delete(mapData, __CHIP_ID_LIST_KEY__)
	delete(mapData, __CHIP_INDEX_LIST_KEY__)
	delete(mapData, __UTILIZATION_APU_CHIPS_KEY__)
	delete(mapData, __UTILIZATION_CPU_CHIPS_KEY__)
	delete(mapData, __UTILIZATION_MEMORY_CHIPS_KEY__)
	delete(mapData, __UTILIZATION_VIC_CHIPS_KEY__)
	delete(mapData, __UTILIZATION_IPE_FPS_CHIPS_KEY__)
	delete(mapData, __TEMPERATURE_CURRENT_CHIPS_KEY__)
	delete(mapData, __VOLTAGE_CURRENT_CHIPS_KEY__)
	delete(mapData, __ECC_MODE_CURRENT_CHIPS_KEY__)
	delete(mapData, __ECC_ERRORS_CORRECTED_TOTAL_CHIPS_KEY__)
	delete(mapData, __ECC_ERRORS_UNCORRECTED_TOTAL_CHIPS_KEY__)
	delete(mapData, __PCI_CHIPS__KEY__)
	delete(mapData, __CLOCKS_KEY__)
	log.Debug("mapData", mapData)
	return flatMapDataToFlatDataStringMapData(mapData)
}

func getMapDataKeyIndex(keyName string, index int) string {
	return keyName + strconv.Itoa(index)
}

func getMapDataSliceValByKeyName(keyName string, mapData map[string]interface{}) []interface{} {
	return mapData[keyName].([]interface{})
}

func getMapDataMapValByKeyName(keyName string, mapData map[string]interface{}) map[string]interface{} {
	return mapData[keyName].(map[string]interface{})
}

func getMapDataDeepMapValByKeyName(keyName string, index int, sliceData []interface{}) interface{} {
	return sliceData[index].(map[string]interface{})[keyName]
}

func getMapDataMapSliceValByKeyName(keyName string, index int, mapData map[string]interface{}) interface{} {
	return mapData[keyName].([]interface{})[index]
}
