package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/extemporalgenome/slug"
	"github.com/gorilla/context"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/mgo.v2/bson"
)

func userget(r *http.Request) (User, error) {
	u := context.Get(r, "User")
	var user User
	err := mapstructure.Decode(u, &user)
	if err != nil {
		return user, err
	}
	return user, nil

}

func (c *appContext) xxxxx(w http.ResponseWriter, r *http.Request) {
	c.xmain()
}
func (c *appContext) adminHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)
	//log.Println(user.Name)
	//log.Println(user.ID)

	count, err := c.db.C("users").Find(bson.M{}).Count()
	if err != nil {
		log.Println(err)
	}

	data := struct {
		User      User
		UserCount int
	}{
		User:      user,
		UserCount: count,
	}
	renderTemplate(w, "admin.html", data)
}

func (c *appContext) backSearchPlace(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)

	r.ParseForm()
	l := strings.Join(r.Form["l"], "")
	s := strings.Join(r.Form["s"], "")
	domain := c.domain + "/api/web/places?l=" + l + "&s=" + s
	data := struct {
		URL       string
		User      User
		MainTitle string
		SubTitle  string
	}{
		URL:       domain,
		User:      user,
		MainTitle: "Places",
		SubTitle:  "search",
	}

	renderTemplate(w, "list.html", data)
}

func (c *appContext) featuredPlaces(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)

	domain := c.domain + "/api/web/places?f=true"
	data := struct {
		URL       string
		User      User
		MainTitle string
		SubTitle  string
	}{
		URL:       domain,
		User:      user,
		MainTitle: "Places",
		SubTitle:  "featured",
	}

	renderTemplate(w, "list.html", data)
}

func (c *appContext) unapprovedPlaces(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)

	domain := c.domain + "/api/web/places?ua=true"
	data := struct {
		URL       string
		User      User
		MainTitle string
		SubTitle  string
	}{
		URL:       domain,
		User:      user,
		MainTitle: "Places",
		SubTitle:  "unapproved",
	}

	renderTemplate(w, "list.html", data)
}
func (c *appContext) allPlaces(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)

	domain := c.domain + "/api/web/places?b=0"
	data := struct {
		URL       string
		User      User
		MainTitle string
		SubTitle  string
	}{
		URL:       domain,
		User:      user,
		MainTitle: "Places",
		SubTitle:  "all",
	}

	renderTemplate(w, "list.html", data)
}
func (c *appContext) adminEditPlace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		user, _ := userget(r)

		URL := strings.Split(r.URL.Path, "/")
		slug := URL[len(URL)-1]
		log.Println(URL)
		log.Println(slug)

		var result place
		err := c.db.C("places").Find(bson.M{"slug": slug}).One(&result)
		if err != nil {
			log.Println(err)
		}
		log.Println(result)
		data := struct {
			Place     place
			User      User
			MainTitle string
			SubTitle  string
		}{
			Place:     result,
			User:      user,
			MainTitle: "Places",
			SubTitle:  "new",
		}
		renderTemplate(w, "singleplace.html", data)
	case "POST":
		log.Println("posting form")
		c.updatePlace(w, r)
		http.Redirect(w, r, r.URL.Path, http.StatusFound)

	}
}

func (c *appContext) newplaceHandler(w http.ResponseWriter, r *http.Request) {

	//log.Println(user.Name)
	switch r.Method {
	case "GET":
		user, _ := userget(r)
		data := struct {
			User      User
			MainTitle string
			SubTitle  string
		}{
			User:      user,
			MainTitle: "Places",
			SubTitle:  "new",
		}
		renderTemplate(w, "newplace.html", data)
	case "POST":
		err := c.newPlace(w, r, 1)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/admin/places/new?q=success", 301)
	}
}

func (c *appContext) newPlace(w http.ResponseWriter, r *http.Request, approve int) error {

	user, err := userget(r)
	if err != nil {
		log.Println(err)
	}
	log.Println(user)
	err = r.ParseMultipartForm(20000)
	if err != nil {
		log.Println(err)
		return err
	}

	f := r.MultipartForm

	files := f.File["images"] // grab filenames

	log.Println(files)

	var img = make([]Images, 0, 10)

	for _, v := range files {
		fn, tn := c.uploadPic(v)
		i := Images{
			Thumb: tn,
			Full:  fn,
		}

		img = append(img, i)

	}

	location := strings.ToLower(strings.Join(r.Form["location"], ""))

	budget := strings.Join(r.Form["budget"], "")

	service := strings.Split(strings.ToLower(strings.Join(r.Form["service"], "")), ",")

	Place := &place{
		Approved:     approve,
		Location:     location,
		Budget:       budget,
		Service:      service,
		State:        strings.Join(r.Form["state"], ""),
		Name:         strings.Join(r.Form["name"], ""),
		Address:      strings.Join(r.Form["address"], ""),
		Overview:     strings.Join(r.Form["overview"], ""),
		WorkingHours: strings.Join(r.Form["working-hours"], ""),
		Phone:        strings.Join(r.Form["phone"], ""),
		PriceRange:   strings.Join(r.Form["price-range"], ""),
		Owner:        user.ID,
		Timestamp:    time.Now(),
		Images:       img[:],
	}

	Place.Slug = slug.Slug(Place.Name + " " + Place.Location + " " + randSeq(5))

	cc := c.db.C("places")
	err = cc.Insert(Place)

	if err != nil {
		log.Println(err)
		return err
	}
	for _, s := range service {
		log.Println(s)
		c.redis.SAdd("services", strings.Title(s))
	}

	c.redis.SAdd("places", strings.Title(Place.Location+", "+Place.State))

	return nil

}

func (c *appContext) updatePlace(w http.ResponseWriter, r *http.Request) error {
	URL := strings.Split(r.URL.Path, "/")
	s := URL[len(URL)-1]
	log.Println(URL)
	log.Println(s)

	err := r.ParseMultipartForm(20000)
	if err != nil {
		log.Println(err)
		return err
	}

	f := r.MultipartForm

	files := f.File["images"] // grab filenames

	log.Println(files)

	var img = make([]Images, 0, 10)

	for _, v := range files {
		fn, tn := c.uploadPic(v)
		i := Images{
			Thumb: tn,
			Full:  fn,
		}

		img = append(img, i)

	}

	location := strings.ToLower(strings.Join(r.Form["location"], ""))
	log.Println(location)
	budget := strings.Join(r.Form["budget"], "")
	log.Println(budget)

	service := strings.Split(strings.ToLower(strings.Join(r.Form["service"], "")), ",")
	log.Println(service)
	log.Println(img)

	cc := c.db.C("places")
	err = cc.Update(bson.M{"slug": s}, bson.M{
		"$set": bson.M{
			"location":     location,
			"budget":       budget,
			"service":      service,
			"state":        strings.Join(r.Form["state"], ""),
			"name":         strings.Join(r.Form["name"], ""),
			"address":      strings.Join(r.Form["address"], ""),
			"overview":     strings.Join(r.Form["overview"], ""),
			"workinghours": strings.Join(r.Form["working-hours"], ""),
			"phone":        strings.Join(r.Form["phone"], ""),
		},
		"$push": bson.M{
			"images": bson.M{
				"$each": img[:],
			},
		},
	})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil

}

func (c *appContext) featurePlace(w http.ResponseWriter, r *http.Request) {
	place := r.URL.Query().Get("slug")
	log.Print("feature ")
	log.Print(place)

	cc := c.db.C("places")
	err := cc.Update(bson.M{"slug": place}, bson.M{
		"$bit": bson.M{
			"featured": bson.M{"xor": 1},
		},
	})

	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

func (c *appContext) approvePlace(w http.ResponseWriter, r *http.Request) {
	place := r.URL.Query().Get("slug")
	log.Print("approve ")
	log.Print(place)

	cc := c.db.C("places")
	err := cc.Update(bson.M{"slug": place}, bson.M{
		"$bit": bson.M{
			"approved": bson.M{"xor": 1},
		},
	})

	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}
