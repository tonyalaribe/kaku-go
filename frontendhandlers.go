package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

//Handlers
func (c *appContext) loginForm(w http.ResponseWriter, r *http.Request) {

	data := struct {
		Login string
	}{
		Login: c.login,
	}
	renderTemplate(w, "login.html", data)
}

func (c *appContext) indexHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	var featured []place
	err := c.db.C("places").Find(bson.M{"featured": 1}).Limit(4).All(&featured)

	if err != nil {
		log.Println(err)
	}

	var x []place

	for _, v := range featured {

		rc := 1

		if v.ReviewsCount > 0 {
			rc = v.ReviewsCount
		}

		rating := v.TotalReviews / rc
		v.Rating = rating

		x = append(x, v)
	}
	var popular []place
	err = c.db.C("places").Find(bson.M{}).Limit(4).Sort("-totalreviews").All(&popular)

	if err != nil {
		log.Println(err)
	}

	var y []place
	for _, v := range popular {

		rc := 1

		if v.ReviewsCount > 0 {
			rc = v.ReviewsCount
		}

		rating := v.TotalReviews / rc
		v.Rating = rating

		y = append(y, v)
	}

	data := struct {
		Featured []place
		Popular  []place
		User     User
		Login    string
		Count    int
	}{
		Featured: x,
		Popular:  y,
		User:     user,
		Login:    c.login,
	}
	renderTemplate(w, "index.html", data)
}

func (c *appContext) searchResultPage(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	//log.Println(user)

	r.ParseForm()
	l := strings.Split(strings.ToLower(strings.Join(r.Form["l"], "")), ",")[0]
	s := strings.Join(r.Form["s"], "")
	b := strings.Join(r.Form["b"], "")

	domain := c.domain + "/api/web/places?l=" + l + "&s=" + s + "&b=" + b
	log.Print(domain)
	count, err := c.SearchResultCount(l, b, s)
	if err != nil {
		log.Println(err)
	}
	data := struct {
		URL      string
		User     User
		Login    string
		Count    int
		Location string
		Budget   string
		Service  string
	}{
		URL:      domain,
		User:     user,
		Login:    c.login,
		Count:    count,
		Location: l,
		Budget:   b,
		Service:  s,
	}

	renderTemplate(w, "search-results.html", data)
}

func (c *appContext) singlePlace(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)
	if r.Method == "GET" {

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

		var reviews []review
		err = c.db.C("reviews").Find(bson.M{"postslug": slug}).All(&reviews)
		rc := 1
		if result.ReviewsCount > 0 {
			rc = result.ReviewsCount
		}

		rating := result.TotalReviews / rc
		result.Rating = rating
		data := struct {
			Place   place
			User    User
			Login   string
			Reviews []review
			Here    string
		}{
			Place:   result,
			User:    user,
			Login:   c.login,
			Reviews: reviews,
			Here:    c.domain + r.URL.String(),
		}
		renderTemplate(w, "single.html", data)
	} else if r.Method == "POST" {

		log.Println("POSTED review")
		err := r.ParseMultipartForm(20000)
		if err != nil {
			log.Println(err)
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
		log.Println(img)
		rate := r.FormValue("rating")
		log.Println(rate)

		rev := r.FormValue("description")
		log.Println(rev)

		s, err := strconv.Atoi(rate)
		if err != nil {
			log.Println(err)
		}

		URL := strings.Split(r.URL.Path, "/")
		slug := URL[len(URL)-1]
		log.Println(URL)
		log.Println(slug)

		rr := review{
			Timestamp:  time.Now(),
			Email:      user.Email,
			Name:       user.Name,
			ProfilePic: user.Image,
			Review:     rev,
			Rating:     s,
			PostSlug:   slug,
			Images:     img,
		}

		cc := c.db.C("reviews")
		err = cc.Insert(rr)
		if err != nil {
			log.Println(err)
		}

		dd := c.db.C("places")
		data := bson.M{
			"$inc": bson.M{
				"totalreviews": s,
				"reviewscount": 1,
			},
		}

		err = dd.Update(bson.M{"slug": slug}, data)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, r.URL.String(), http.StatusFound)

	}
}

func (c *appContext) AutoCompletePlaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-Type", "application/json")

	p := c.redis.SMembers("places")
	places, err := p.Result()
	if err != nil {
		log.Println(err)
	}
	log.Println(places)
	json.NewEncoder(w).Encode(places)
}

func (c *appContext) AutoCompleteServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-Type", "application/json")

	p := c.redis.SMembers("services")
	services, err := p.Result()
	if err != nil {
		log.Println(err)
	}
	log.Println(services)
	json.NewEncoder(w).Encode(services)
}

func (c *appContext) SearchJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	page := 1
	perPage := 12
	r.ParseForm()

	log.Println(len(strings.Join(r.Form["page"], "")))
	if len(strings.Join(r.Form["page"], "")) > 0 {

		a, _ := strconv.Atoi(strings.Join(r.Form["page"], ""))
		page = a + 1
		log.Println("length was greater")
	}

	log.Println(page)

	location := strings.ToLower(strings.Join(r.Form["l"], ""))
	budget := strings.ToLower(strings.Join(r.Form["b"], ""))
	query := strings.ToLower(strings.Join(r.Form["s"], ""))
	var unapproved bool
	var featured bool
	if strings.Join(r.Form["ua"], "") == "true" {
		unapproved = true
	}
	if strings.Join(r.Form["f"], "") == "true" {
		featured = true
	}
	loc := strings.Split(location, ",")
	Results, err := c.SearchResult(loc[0], budget, query, unapproved, featured, page, perPage)
	if err != nil {
		log.Println(err)
	}

	var z []place

	for _, v := range Results {

		rc := 1

		if v.ReviewsCount > 0 {
			rc = v.ReviewsCount
		}

		rating := v.TotalReviews / rc
		v.Rating = rating
		z = append(z, v)
	}

	//log.Println(Results)
	x, err := json.Marshal(z)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(x)

}

func (c *appContext) userHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		/*		r.ParseForm()
						data := &user{}
						err := json.NewDecoder(r.Body).Decode(data)
						if err != nil {
							log.Println(err)
						}

						if data.Provider == "local" {
							id, err := c.newLocalAuth(data.Email, data.Password)
							if err != nil {
								log.Println(err)
							}
							data.UserID = id

						}

						ID, err := c.newUser(data)
						if err != nil {
							log.Println(err)
						}

				log.Println(ID)
		*/
	}

}
