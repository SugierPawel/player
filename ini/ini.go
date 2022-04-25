package ini

import (
	"flag"
	"log"
	"os"

	"github.com/SugierPawel/news/rpc/core"
	"github.com/go-ini/ini"
)

var iniFIleFlag = flag.String("conf", "./conf.ini", "Brak pliku konfiguracyjnego.")
var IniConfig *ini.File
var SCMap map[string]*core.StreamConfig
var StreamConfigChan = make(chan *core.StreamConfig, 1)

func WriteSection(sc *core.StreamConfig) {
	if _, ok := SCMap[sc.StreamName]; ok {
		log.Printf("Strumień: %s, już istnieje: %v\n", sc.StreamName, sc)
	} else {
		IniConfig.NewSection(sc.StreamName)
		if err := IniConfig.Section(sc.StreamName).ReflectFrom(&sc); err != nil {
			log.Printf("Błąd zapisu: %s, %s", *iniFIleFlag, err)
		}
		IniConfig.SaveTo(*iniFIleFlag)
		SCMap[sc.StreamName] = sc
		sc.Request = "AddRTPsource"
		StreamConfigChan <- sc
	}
}
func DeleteSection(sc *core.StreamConfig) {
	if _, ok := SCMap[sc.StreamName]; ok {
		sc.Request = "DelRTPsource"
		StreamConfigChan <- sc
		delete(SCMap, sc.StreamName)
		IniConfig.DeleteSection(sc.StreamName)
		IniConfig.SaveTo(*iniFIleFlag)
	} else {
		log.Printf("Strumień: %s, nie istnieje", sc.StreamName)
	}
}
func ReadIniConfig() {
	flag.Parse()
	SCMap = make(map[string]*core.StreamConfig)
	if *iniFIleFlag == "" {
		log.Printf("Błąd ładowania: %s", *iniFIleFlag)
		return
	}
	if _, err := os.Stat(*iniFIleFlag); os.IsNotExist(err) {
		log.Printf("Tworzę plik konfiguracyjny: %s", *iniFIleFlag)
		f, err := os.Create(*iniFIleFlag)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}
	var err error
	IniConfig, err = ini.Load(*iniFIleFlag)
	if err != nil {
		log.Printf("Błąd odczytu: %s", *iniFIleFlag)
		os.Exit(0)
	}
	for _, section := range IniConfig.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		SCMap[section.Name()] = new(core.StreamConfig)
		err = IniConfig.Section(section.Name()).MapTo(SCMap[section.Name()])
		if err != nil {
			log.Printf("Błąd mapowania sekcji: [%s] -> %v\n", section.Name(), SCMap[section.Name()])
			continue
		}
		SCMap[section.Name()].Request = core.AddRTPsourceRequest
		SCMap[section.Name()].StreamName = section.Name()
		log.Printf("Dodaję sekcję: [%s] -> %v\n", section.Name(), SCMap[section.Name()])
	}
	for _, sc := range SCMap {
		StreamConfigChan <- sc
	}
}
