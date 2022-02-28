package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"lynxi_smi_pro/internal/exporter"
	"strconv"
)

var (
	query    = kingpin.Flag("query", "Display APU or hardware Detail info.").Short('q').Bool()
	board_id = kingpin.Flag("index", "Target a specific Board index. This Flag is used to query APU or hardware Detail details").Short('i').String()
	chip_id  = kingpin.Flag("chip_id", "Target a specific Chip ID. This Flag is used to query APU or hardware Detail details").Short('c').String()
	//type_info      = kingpin.Flag("type", "Show information for type: board,memory, usages,temp, power, volt, ecc-enable, health, product, ecc.").Short('t').PlaceHolder("board").String()
	query_apu      = kingpin.Flag("query-apu", "Query Information about APU.").PlaceHolder("name,driver_version,power,...").String()
	list_apus      = kingpin.Flag("list-apus", "Display a list of APUs connected to the system.").Short('L').Bool()
	chip_count     = kingpin.Flag("chip-count", "Displays the number of KA200.").Bool()
	chip_list      = kingpin.Flag("chip-list", "Displays a list of KA200.").Bool()
	debug          = kingpin.Flag("debug", "Display Debug Info").Bool()
	help_query_apu = kingpin.Flag("help-query-apu", "Display Help Query Information about APU.").Bool()
)

func main() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{
		PrettyPrint: true,
	})
	kingpin.HelpFlag.Short('h')
	kingpin.UsageTemplate(kingpin.SeparateOptionalFlagsUsageTemplate)
	kingpin.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	switch {
	case *query:
		switch {
		case *board_id != "" && *chip_id != "":
			boardIndex, err := strconv.Atoi(*board_id)
			if err != nil {
				kingpin.Errorf("board index (%s)is not int value", *board_id)
			}
			chipId, err := strconv.Atoi(*chip_id)
			if err != nil {
				kingpin.Errorf("chip id (%s)is not int value", *chip_id)
			}
			if boardIndex < 0 && chipId < 0 {
				kingpin.Errorf("board index And chip id must be greater than 0.")
			} else {
				exporter.QueryLynSmiDetailInfoByChipIDAndBoardId(&boardIndex, &chipId)
			}
		case *board_id != "":
			boardIndex, err := strconv.Atoi(*board_id)
			if err != nil {
				kingpin.Errorf("board index (%s)is not int value", *board_id)
			}
			if boardIndex < 0 {
				kingpin.Errorf("board index must be greater than 0.")
			} else {
				exporter.QueryLynSmiDetailInfoByBoardId(&boardIndex)
			}
		case *chip_id != "":
			kingpin.Errorf("board index is requested")
		default:
			exporter.QueryLynSmiDetailInfo()
		}
	case *list_apus:
		exporter.ListAPUs()
	case len(*query_apu) > 0:
		switch {
		case len(*query_apu) < 0:
			kingpin.Usage()
		}
		exporter.QueryLynAPUsInfo(*query_apu)
	case *chip_count:
		exporter.QueryLynChipTotalNum()
	case *chip_list:
		exporter.QueryLynChipList()
	case *help_query_apu:
		exporter.QueryAPUHelpInfo()
	case *board_id != "" || *chip_id != "":
		kingpin.Usage()
	default:
		exporter.QueryLynSmiInfo()
	}
}
