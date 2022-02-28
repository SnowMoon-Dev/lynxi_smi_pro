package exporter

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"strconv"
	"strings"
	"time"
)

func QueryAPUHelpInfo() {
	fmt.Print("List of valid properties to query for the switch query-apu:\n\n")
	for _, v := range fallbackQField {
		fmt.Println(strconv.Quote(string(v)))
		fmt.Println(fallbackQFieldComment[v])
		fmt.Println()
	}
}

func QueryLynSmiInfo() {
	fn := func(line string) {
		line = replaceNAInfo(line)
		line = replaceProductNameToSmiVersionName(line)
		line = replaceDriverVersionInfo(line)
		fmt.Print(line)
	}
	RunLynSMICmdAndReadStrings(fn)
}

func QueryLynAPUsInfo(qFieldsRaw string) {
	var (
		boardBaseInfo     BoardBaseInfo
		boardBaseInfoList []BoardBaseInfo
	)
	qFields, err := verifyAndCheckQueryFields(qFieldsRaw)
	if err != nil {
		return
	}
	r, cmd := runLynSMIDetailCommand()
	line, err := r.ReadString(__LINE_FEED_SEP__)
	ch := make(chan []string)
	go QueryLynPciInfo(ch)
	pciInfoStrList := <-ch
	fn := func(info string) {
		switch {
		case strings.Contains(info, __BOARD_STR__):
			now := time.Now()
			timestamp := now.Unix()
			boardBaseInfo.TimeStamp = strconv.FormatInt(timestamp, 10)
			boardBaseInfo.BoardIndex = getBoardInfoVal(info)
		case strings.Contains(info, __PRODUCT_NAME_STR__):
			boardBaseInfo.ProductName = getBoardInfoVal(info)
		case strings.Contains(info, __PRODUCT_BRAND_STR__):
			boardBaseInfo.ProductBrand = getBoardInfoVal(info)
		case strings.Contains(info, __PRODUCT_NUMBER_STR__):
			boardBaseInfo.ProductNumber = getBoardInfoVal(info)
		case strings.Contains(info, __DRIVER_STR__):
			boardBaseInfo.DriverVersion = strings.Split(getVersion(Driver), __V_SEP__)[1]
		case strings.Contains(info, __FIRMWARE_VERSION_STR__):
			boardBaseInfo.FirmwareVersion = getBoardInfoVal(info)
		case strings.Contains(info, __SERIAL_NUMBER_STR__):
			boardBaseInfo.SerialNumber = getBoardInfoVal(info)
		case strings.Contains(info, __CHIP_COUNT_STR__):
			boardBaseInfo.ChipCount = getBoardInfoVal(info)
		case strings.Contains(info, __CHIP_ID_STR__):
			boardBaseInfo.ChipIdList, boardBaseInfo.ChipCount = getChipIdListByChipCount(r, &info)
			boardIndex, _ := strconv.Atoi(boardBaseInfo.BoardIndex)
			chipIndex, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			chipStartIndex := computeCurrentChipIndex(boardIndex, chipIndex)
			boardBaseInfo.ChipIndexList = getChipIndexByChipCountAndBoardIndex(chipStartIndex, boardBaseInfo.ChipCount)

			if strings.Contains(info, __UUID_STR__) || strings.Contains(info, __ECID_STR__) {
				count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
				uuidList := make([]string, count)
				line, _ := r.ReadString(__LINE_FEED_SEP__)
				for i := 0; i < count; i++ {
					uuidList[i] = getBoardInfoVal(line)
					line, _ = r.ReadString(__LINE_FEED_SEP__)
				}
				boardBaseInfo.UuidList = uuidList
			}
		case strings.Contains(info, __APU_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				boardBaseInfo.ApuTotal = getBoardUtilTotalVal(line)
				boardBaseInfo.ApuUtilList = getBoardUtilInfo(line, boardBaseInfo.ChipCount, r)
			}
		case strings.Contains(info, __CPU_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				boardBaseInfo.CpuTotal = getBoardUtilTotalVal(line)
				boardBaseInfo.CpuUtilList = getBoardUtilInfo(line, boardBaseInfo.ChipCount, r)
			}
		case strings.Contains(info, __VIC_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				boardBaseInfo.VicTotal = getBoardUtilTotalVal(line)
				boardBaseInfo.VicUtilList = getBoardUtilInfo(line, boardBaseInfo.ChipCount, r)
			}
		case strings.Contains(info, __MEMORY_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				boardBaseInfo.MemoryTotal = getBoardUtilTotalVal(line)
				boardBaseInfo.MemoryUtilList = getBoardUtilInfo(line, boardBaseInfo.ChipCount, r)
			}
		case strings.Contains(info, __IPE_FPS_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				boardBaseInfo.IpeTotal = getBoardInfoVal(line)
				boardBaseInfo.IpeUtilList = getBoardUtilInfo(line, boardBaseInfo.ChipCount, r)
			}
		case !strings.Contains(info, __PCIE_GEN_STR__) && !strings.Contains(info, __PCIE_SWITCH__) && strings.Contains(info, __PCI_STR__) || strings.Contains(info, __PCIE_STR__):
			_boardIndex, _ := strconv.Atoi(boardBaseInfo.BoardIndex)
			_chipCount, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			chipIndex := computeCurrentChipIndex(_boardIndex, _chipCount)
			boardPciDeviceInfoList := make([]BoardPciDeviceInfo, _chipCount)
			for i := 0; i < _chipCount; i++ {
				var boardPciDeviceInfo BoardPciDeviceInfo
				pciChipIndex := i
				vendorId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, VendorId)
				vendorId = removeLineBreak(vendorId)
				deviceId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, DeviceId)
				deviceId = removeLineBreak(deviceId)
				subVendorId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, SubVendorId)
				subVendorId = removeLineBreak(subVendorId)
				subDeviceId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, SubDeviceId)
				subDeviceId = removeLineBreak(subDeviceId)
				busNum, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, BusNum)
				busNum = removeLineBreak(busNum)
				device, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, Device)
				device = removeLineBreak(device)
				function, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, Function)
				function = removeLineBreak(function)
				maxSpeed, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, MaxSpeed)
				maxSpeed = removeLineBreak(maxSpeed)
				maxWidth, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, MaxWidth)
				maxWidth = removeLineBreak(maxWidth)
				currentSpeed, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, CurrentSpeed)
				currentSpeed = removeLineBreak(currentSpeed)
				currentWidth, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, CurrentWidth)
				currentWidth = removeLineBreak(currentWidth)
				numaNodeId, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, NumaNodeId)
				numaNodeId = removeLineBreak(numaNodeId)
				numaCpuList, _ := getPciDeviceInfo(pciChipIndex, chipIndex, pciInfoStrList, NumaCPUList)
				numaCpuList = removeLineBreak(numaCpuList)
				if strings.Contains(numaCpuList, __COMMA_SEP__) {
					numaCpuList = strings.Replace(numaCpuList, __COMMA_SEP__, __SPCAE_SEP__, -1)
				}
				log.WithFields(log.Fields{
					__PRODUCT_NAME_STR__: boardBaseInfo.ProductName,
					__CHIP_COUNT_STR__:   boardBaseInfo.ChipCount,
					__VENDOR_ID_STR__:    vendorId,
					__DEVICE_ID_STR__:    deviceId,
					SubSystemVendor:      subVendorId,
					SubSystemDevice:      subDeviceId,
					__BUS_NUM__:          busNum,
					__DEVICE__:           device,
					__FUNCTION__:         function,
					CurrentLinkSpeed:     currentSpeed,
					CurrentLinkWidth:     currentWidth,
					MaxLinkSpeed:         maxSpeed,
					MaxLinkWidth:         maxWidth,
					NumaNode:             numaNodeId,
					NumaNodeCPUList:      numaCpuList,
				}).Debug("Board Summery Info")
				boardPciDeviceInfo.Set(vendorId, deviceId, subVendorId, subDeviceId, busNum,
					device, function, maxSpeed, maxWidth, currentSpeed,
					currentWidth, numaNodeId, numaCpuList)
				boardPciDeviceInfoList[pciChipIndex] = boardPciDeviceInfo
			}
			boardBaseInfo.PciInfoList = boardPciDeviceInfoList
		case strings.Contains(info, __FAN_STR__):
			boardBaseInfo.FanSpeed = getBoardInfoVal(info)
			if strings.Contains(boardBaseInfo.FanSpeed, __NA_STR__) {
				boardBaseInfo.FanSpeed = __N_A_STR__
			}
		case strings.Contains(info, __TEMPERATURE_STR__):
			count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			tempList := make([]string, count)
			for i := 0; i < count; i++ {
				line, _ := r.ReadString(__LINE_FEED_SEP__)
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				tempList[i] = getBoardInfoVal(line)
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				line, _ = r.ReadString(__LINE_FEED_SEP__)
			}
			boardBaseInfo.TempUtilList = tempList
		case !strings.Contains(info, __BOARD_VOLTAGE_STR__) && strings.Contains(info, __VOLTAGE_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			chipVoltageList := make([]string, count)
			for i := 0; i < count; i++ {
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				chipVoltageList[i] = getBoardInfoVal(line)
			}
			boardBaseInfo.ChipVoltageList = chipVoltageList
		case strings.Contains(info, __CLOCKS_STR__):
			var boardClocksInfo BoardClocksInfo
			count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			apuClocksList := make([]string, count)
			apuClocksMaxList := make([]string, count)
			cpuClocksList := make([]string, count)
			cpuClocksMaxList := make([]string, count)
			memoryClocksList := make([]string, count)
			memoryClocksMaxList := make([]string, count)
			for i := 0; i < count; i++ {
				_, _ = r.ReadString(__LINE_FEED_SEP__)
				getClocksVal(r, apuClocksList, i)
				getClocksVal(r, apuClocksMaxList, i)
				getClocksVal(r, cpuClocksList, i)
				getClocksVal(r, cpuClocksMaxList, i)
				getClocksVal(r, memoryClocksList, i)
				getClocksVal(r, memoryClocksMaxList, i)
			}
			boardClocksInfo.ApuClocksList = apuClocksList
			boardClocksInfo.ApuClocksMaxList = apuClocksMaxList
			boardClocksInfo.CpuClocksList = cpuClocksList
			boardClocksInfo.CpuClocksMaxList = cpuClocksMaxList
			boardClocksInfo.MemoryClocksList = memoryClocksList
			boardClocksInfo.MemoryClocksMaxList = memoryClocksMaxList
			boardBaseInfo.ClocksInfo = boardClocksInfo
		case strings.Contains(info, __BOARD_VOLTAGE_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __INPUT_STR__) {
				boardBaseInfo.VoltageInput = getBoardInfoVal(line)
			}
		case strings.Contains(info, __POWER_LIMIT_STR__):
			boardBaseInfo.PowerLimit = getBoardInfoVal(line)
		case strings.Contains(info, __POWER_DRAW_STR__):
			boardBaseInfo.PowerDraw = getBoardInfoVal(line)
		case strings.Contains(info, __ECC_MODE_STR__):
			count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
			eccModeList := make([]string, count)
			for i := 0; i < count; i++ {
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				eccModeList[i] = getBoardInfoVal(line)
			}
			boardBaseInfo.EccModeList = eccModeList
		case strings.Contains(info, __DDR_ECC_ERR_COUNT_STR__):
			line, _ := r.ReadString(__LINE_FEED_SEP__)
			if strings.Contains(line, __TOTAL_STR__) {
				line, _ := r.ReadString(__LINE_FEED_SEP__)
				if strings.Contains(line, __CORRECTED_ERR_STR__) {
					boardBaseInfo.DdrEccCorrectedTotal = getBoardInfoVal(line)
				}
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				if strings.Contains(line, __UNCORRECTED_ERR_STR__) {
					boardBaseInfo.DdrEccUnCorrectedTotal = getBoardInfoVal(line)
				}
				line, _ = r.ReadString(__LINE_FEED_SEP__)
				count, _ := strconv.Atoi(boardBaseInfo.ChipCount)
				ddrEccErrCorrectedChipCount := make([]string, count)
				ddrEccErrUnCorrectedChipCount := make([]string, count)
				if strings.Contains(line, __CHIP_STR__) {
					for i := 0; i < count; i++ {
						line, _ = r.ReadString(__LINE_FEED_SEP__)
						ddrEccErrCorrectedChipCount[i] = getBoardInfoVal(line)
						line, _ = r.ReadString(__LINE_FEED_SEP__)
						ddrEccErrUnCorrectedChipCount[i] = getBoardInfoVal(line)
						line, _ = r.ReadString(__LINE_FEED_SEP__)
					}
					boardBaseInfo.DdrEccErrCorrectedChipCount = ddrEccErrCorrectedChipCount
					boardBaseInfo.DdrEccErrUnCorrectedChipCount = ddrEccErrUnCorrectedChipCount
				} else {
					boardBaseInfo.DdrEccErrCorrectedChipCount = ddrEccErrCorrectedChipCount
					boardBaseInfo.DdrEccErrUnCorrectedChipCount = ddrEccErrUnCorrectedChipCount
				}
				boardBaseInfoList = append(boardBaseInfoList, boardBaseInfo)
				boardBaseInfo = BoardBaseInfo{}
			}
		}
	}
	for err == nil {
		line = removeAndReplaceBaseInfo(line)
		fn(line)
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
	err = cmd.Wait()
	if err != nil {
		log.Debugln(err)
	}
	log.Debugf("Board Info Number: %d", len(boardBaseInfoList))
	printAPUsInfoTitle(qFields)
	for _, v := range boardBaseInfoList {
		mapData := structToMap(v)
		boardBaseInfoStrMap := apusInfoToFlatMap(mapData)
		for i, qField := range qFields {
			val := getAPUInfoByBoardMapInfo(boardBaseInfoStrMap, qField)
			if len(qFields)-1 != i {
				fmt.Print(val + __COMMA_SEP__ + __SPCAE_SEP__)
			} else {
				fmt.Print(val)
			}
		}
		fmt.Println()
	}
}

func ListAPUs() {
	var boardIndex int = 0
	var chipCount int = 0
	var boardSN string
	var boardPN string
	r, cmd := runLynSMIDetailCommand()
	line, err := r.ReadString(__LINE_FEED_SEP__)
	removeDebugInfo(&line, r, err)
	for err == nil {
		line = removeAndReplaceBaseInfo(line)
		boardIndexStr, errCode := getBoardIndex(line)
		if errCode != -1 {
			boardIndex, _ = strconv.Atoi(boardIndexStr)
		}
		pn, errCode := getBoardPN(line)
		if errCode != -1 {
			boardPN = pn
		}
		if !isStrBlank(line) {
			if strings.Contains(line, __CHIP_ID_STR__) {
				chipCount = getChipCountByChipId(r, &line)
			}
			sn, errCode := getBoardSN(line)
			if errCode != -1 {
				boardSN = sn
			}
			if strings.Contains(line, __UTILIZATION_STR__) {
				fmt.Println(__APU_STR__ + __SPCAE_SEP__ + strconv.Itoa(boardIndex) + __COLON_SEP__ + boardPN + "  (" + "SN: " + boardSN + __COMMA_SEP__ + " ChipCount: " + strings.TrimSpace(strconv.Itoa(chipCount)) + ")")
			}
		}
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
	err = cmd.Wait()
	if err != nil {
		log.Debugln(err)
		return
	}
}

func QueryLynSmiDetailInfoByChipIDAndBoardId(boardId *int, chipId *int) {
	r, cmd := runLynSMICommandByChipIDAndBoardId(boardId, chipId)
	getLynSmiDetailInfo(r, cmd)
}

func QueryLynSmiDetailInfoByBoardId(boardId *int) {
	r, cmd := runLynSMICommandByBoardId(boardId)
	getLynSmiDetailInfo(r, cmd)
}

func QueryLynSmiDetailInfo() {
	r, cmd := runLynSMIDetailCommand()
	getLynSmiDetailInfo(r, cmd)
}

func QueryLynPciInfo(ch chan<- []string) {
	var pciInfoStrList []string
	fn := func(line string) {
		pciInfoStrList = append(pciInfoStrList, string(strings.Split(line, " ")[0]))
	}
	RunShellCmdAndArgsAndReadString(fn, DefaultPCICommand, LynChipDeviceIDAndVendorIDCmdParam...)
	ch <- pciInfoStrList
}

func QueryLynChipTotalNum() {
	var chip_count int = 0
	fn := func(line string) {
		chip_count++
	}
	RunShellCmdAndArgsAndReadString(fn, DefaultPCICommand, LynChipDeviceIDAndVendorIDCmdParam...)
	fmt.Printf("ChipTotalNumbyPci: %s\n", strconv.Itoa(chip_count))
}

func QueryLynChipList() {
	fn := func(line string) {
		fmt.Print(line)
	}
	RunShellCmdAndArgsAndReadString(fn, DefaultPCICommand, LynChipDeviceIDAndVendorIDCmdParam...)
}
