package main

import (
	"errors"
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//Authenticate check if user exists if not create a new user document NewUser function is called within this function. note the user struct being passed
//to this function should alredi contain a self generated objectid
func (c *appContext) Authenticate(user *User, provider string) (*User, error) {
	log.Println("Authenticate")
	result := User{}
	C := c.db.C("users")

	log.Println(user.PID)
	log.Println(provider)

	//err := C.Find(bson.M{"provideruid": user.PID, "provider": provider}).One(&result)

	user.Provider = "facebook"

	change := mgo.Change{
		Update:    bson.M{"$set":bson.M{
		  "name":user.Name,
		  "email":user.Email,
		  "image":user.Image,
		  "phone":user.Phone,
		  },
		  },
		Upsert:    true,
		ReturnNew: true,
	}
	info, err := C.Find(bson.M{"pid": user.PID, "provider": provider}).Apply(change, &result)
	log.Println(info)
	log.Println(result)

	if err != nil {
		return &result, err
	}
	//if result.Provider != "" {
	//	return &result, nil
	//}

	//return c.NewUser(user, provider)
	return &result, nil
}

//NewUser is for adding a new user to the database. Please note that what you pass to the function is a pointer to the actual data, note the data its self. ie newUser(&NameofVariable)
func (c *appContext) NewUser(data *User, socialProvider string) (*User, error) {

	collection := c.db.C("users")
	data.ID = bson.NewObjectId().Hex()
	data.Provider = socialProvider

	err := collection.Insert(data)
	if err != nil {
		log.Println(err)
		return data, err
	}

	return data, nil
}

//SearchResult returns the list of places based on parameters passed to it, like
//the location, budget, query, string and the page
func (c *appContext) SearchResult(location string, budget string, query string, unapproved bool, featured bool, page int, perPage int) ([]place, error) {
	var Results []place

	d := c.db.C("places")

	index := mgo.Index{
		Key: []string{"$text:name", "$text:service"},
	}

	err := d.EnsureIndex(index)
	if err != nil {
		log.Println(err)
		return Results, err
	}

	var q *mgo.Query
	log.Println("about to build query")
	if budget == "" && location == "" && query != "" {
		q = d.Find(
			bson.M{
				"$text": bson.M{
					"$search": query,
				},
			},
		)

	} else if query == "" && location != "" && budget != "" {

		q = d.Find(
			bson.M{
				"location": location,
				"budget": bson.M{
					"$lte": budget,
				},
			},
		)
	} else if budget != "" && location != "" && query != "" {
		q = d.Find(
			bson.M{
				"location": location,
				"budget": bson.M{
					"$lte": budget,
				},
				"$text": bson.M{
					"$search": query,
				},
			},
		)

	} else if budget == "" && location != "" && query != "" {
		q = d.Find(bson.M{
			"location": location,
			"$text": bson.M{
				"$search": query,
			},
		})
	} else if unapproved {
		q = d.Find(bson.M{"approved": 0})
		log.Println("using approved:1")
	} else if featured {
		log.Println("using featured")
		q = d.Find(bson.M{"featured": 1})
	} else {
		q = d.Find(bson.M{})
		log.Println("in else")
	}
	count, err := q.Count()
	log.Println(count)
	if err != nil {
		log.Println(err)
		return Results, err
	}
	err = hasNext(count, page, perPage)
	if err == nil {
		skip := perPage * (page - 1)
		log.Print("skip")
		log.Print(skip)
		err = q.Limit(perPage).Skip(skip).All(&Results)
		log.Println("got to result stage")
		log.Println(Results)
		if err != nil {
			log.Println(err)
			return Results, err
		}

		return Results, nil

	}

	return Results, nil

}

func hasNext(count, page, perpage int) error {
	var total int
	log.Println(page)

	if count%perpage != 0 {
		total = count/perpage + 1
	} else {
		total = count / perpage
	}
	if total < page {
		page = total
	}

	if total >= page {
		return nil
	}
	return errors.New("No next page")
}

//SearchResult returns the list of places based on parameters passed to it, like
//the location, budget, query, string and the page
func (c *appContext) SearchResultCount(location string, budget string, query string) (int, error) {
	var count int

	d := c.db.C("places")

	index := mgo.Index{
		Key: []string{"$text:name", "$text:service"},
	}

	err := d.EnsureIndex(index)
	if err != nil {
		log.Println(err)
		return count, err
	}

	var q *mgo.Query
	log.Println("about to build query")
	if budget == "" && location == "" && query != "" {
		q = d.Find(
			bson.M{
				"$text": bson.M{
					"$search": query,
				},
			},
		)

	} else if query == "" && location != "" && budget != "" {

		q = d.Find(
			bson.M{
				"location": location,
				"budget": bson.M{
					"$lte": budget,
				},
			},
		)
	} else if budget != "" && location != "" && query != "" {
		q = d.Find(
			bson.M{
				"location": location,
				"budget": bson.M{
					"$lte": budget,
				},
				"$text": bson.M{
					"$search": query,
				},
			},
		)

	} else if budget == "" && location != "" && query != "" {
		q = d.Find(bson.M{
			"location": location,
			"$text": bson.M{
				"$search": query,
			},
		})
	} else {
		q = d.Find(bson.M{})
		log.Println("in else")
	}
	count, err = q.Count()
	log.Println(count)
	if err != nil {
		log.Println(err)
		return count, err
	}

	return count, nil

}
