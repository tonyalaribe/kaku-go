package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/extemporalgenome/slug"
	"gopkg.in/mgo.v2/bson"
)

func (c *appContext) xmain() {
	inFile, err := os.Open("./places.json")
	if err != nil {

		log.Println(err)
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	//var pp []place
	for scanner.Scan() {
		//log.Println(scanner.Text())

		type Placex struct {
			ID           bson.ObjectId `bson:"_id,omitempty"`
			Slug         string
			Location     string
			State        string
			Budget       string
			Service      []string
			Name         string
			Overview     string
			Address      string
			WorkingHours string
			Phone        string
			PriceRange   string
			Owner        string
			//Timestamp    string
			Images []Images
			Rating int
		}
		var p Placex
		err := json.Unmarshal([]byte(scanner.Text()), &p)
		if err != nil {

			log.Println(err)
		}
		//log.Println(p.Images)

		//pp = append(pp, p)

		var services []string
		for _, s := range strings.Split(p.Service[0], " ") {
			log.Println(s)
			c.redis.SAdd("services", strings.Title(strings.ToLower(s)))

			services = append(services, strings.ToLower(s))
		}
		c.redis.SAdd("places", strings.Title(strings.ToLower(p.Location)+", Lagos"))
		xx := &place{
			Location:     strings.ToLower(p.Location),
			State:        "lagos",
			Name:         strings.ToLower(p.Name),
			Budget:       strings.ToLower(p.Budget),
			Service:      services,
			Overview:     p.Overview,
			Address:      p.Address,
			WorkingHours: p.WorkingHours,
			Phone:        p.Phone,
			Timestamp:    time.Now(),
			Images:       p.Images,
			Owner:        p.Owner,
			Slug:         slug.Slug(p.Name + " " + p.Location + " " + randSeq(5)),
		}
		cc := c.db.C("places")
		err = cc.Insert(xx)
		if err != nil {
			log.Println(err)

		}

	}
}
