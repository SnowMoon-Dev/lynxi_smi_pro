package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"lynxi_smi_pro/internal/exporter"
)

var (
	query    = kingpin.Flag("query", "Display APU or hardware Detail info.").Short('q').Bool()
	board_id = kingpin.Flag("index", "Target a specific Board index.").Short('i').Default("-1").Hidden().Int()
	chip_id  = kingpin.Flag("chip_id", "Target a specific Chip ID.").Short('c').Default("-1").Hidden().Int()
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
		case *board_id >= 0 && *chip_id >= 0:
			exporter.QueryLynSmiDetailInfoByChipIDAndBoardId(board_id, chip_id)
		case *board_id >= 0:
			exporter.QueryLynSmiDetailInfoByBoardId(board_id)
		case *chip_id >= 0:
			kingpin.Errorf("board id is requested")
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
	case *board_id >= 0 || *chip_id >= 0:
		kingpin.Usage()
	default:
		exporter.QueryLynSmiInfo()
	}
}
