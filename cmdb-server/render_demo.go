package main

import (
	"encoding/json"
	"fmt"
	"github.com/WeBankPartners/we-cmdb/cmdb-server/common/graph"
	"github.com/WeBankPartners/we-cmdb/cmdb-server/models"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	RenderExample()
}

type viewSettings struct {
	Editable      string              `json:"editable"`
	SuportVersion string              `json:"suportVersion"`
	Multiple      string              `json:"multiple"`
	Report        string              `json:"report"`
	CiType        string              `json:"ciType"`
	FilterAttr    string              `json:"filterAttr"`
	FilterValue   string              `json:"filterValue"`
	Graphs        []models.GraphQuery `json:"graphs"`
}

type CiType struct {
	CiTypeId     string `json:"ciTypeId"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ImageFile    string `json:"imageFile"`
	CiGroup      string `json:"ciGroup"`
	CiLayer      string `json:"ciLayer"`
	CiTemplate   string `json:"ciTemplate"`
	StateMachine string `json:"stateMachine"`
	SeqNo        string `json:"seqNo"`
	Attributes   []struct {
		CiTypeAttrId string `json:"ciTypeAttrId"`
		CiTypeId     string `json:"ciTypeId"`
		Name         string `json:"name"`
	} `json:"attributes"`
}
type CiTypeMap map[string]CiType

func RenderExample() {
	var viewData []map[string]interface{}
	if fileBytes, err := os.ReadFile("viewData.json"); err == nil {
		_ = json.Unmarshal(fileBytes, &viewData)
	} else {
		log.Fatal(err)
	}

	var settingData viewSettings
	if fileBytes, err := os.ReadFile("viewSettings.json"); err == nil {
		_ = json.Unmarshal(fileBytes, &settingData)
	} else {
		log.Fatal(err)
	}

	var ciTypeMapping CiTypeMap
	if fileBytes, err := os.ReadFile("ciTypeMapping.json"); err == nil {
		err = json.Unmarshal(fileBytes, &ciTypeMapping)
	} else {
		log.Fatal(err)
	}

	imageMap := map[string]string{}
	for _, ciType := range ciTypeMapping {
		imageMap[ciType.CiTypeId] = filepath.Join("/wecmdb/fonts/", ciType.ImageFile)
	}

	var err error
	var dot string
	for index, g := range settingData.Graphs {
		//if index == 0 {
		//	continue
		//}
		fmt.Println(index, g.Name, g.ViewGraphType)
		if dot, err = graph.RenderGraph(
			g,
			viewData,
			graph.RenderOption{SuportVersion: settingData.SuportVersion, ImageMap: imageMap},
		); err != nil {
			log.Fatal(err)
		}

		fmt.Println(dot)

		filename := fmt.Sprintf("d3-demo-%d.html", index)
		outputFile, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		renderHtml(outputFile, dot)
	}
}

func renderHtml(wr io.Writer, dot string) {
	tf, err := template.ParseFiles("d3-demo-template.html")
	if err != nil {
		log.Fatal(err)
	}
	err = tf.Execute(wr, map[string]interface{}{
		"Dot": dot,
	})
	if err != nil {
		log.Fatal(err)
	}
}
