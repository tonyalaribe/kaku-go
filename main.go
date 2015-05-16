package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"

	"golang.org/x/oauth2"
	"gopkg.in/mgo.v2"

	"github.com/gorilla/context"
	"github.com/justinas/alice"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"gopkg.in/redis.v2"
)

const (
	//Cost is the well, cost of the bcrypt encryption used for storing user
	//passwords in the database
	Cost int = 5
)

type appContext struct {
	db        *mgo.Database
	verifyKey []byte
	signKey   []byte
	token     string

	login      string
	fbclientid string
	fbsecret   string
	domain     string

	bucket *s3.Bucket
	redis  *redis.Client
}

//pre parse the template files, and store them in memory. Fail if
//they're not found
var templates = template.Must(template.ParseFiles("templates/index.html", "templates/single.html", "templates/search-results.html", "templates/ftmp.html", "templates/admin/admin.html", "templates/admin/newplace.html", "templates/admin/list.html", "templates/admin/singleplace.html", "templates/admin/login.html", "templates/admin/tmp.html"))

//renderTemplate is simply a helper function that takes in the response writer
//interface, the template file name and the data to be passed in, as an
//interface. It causes an internal server error if any of the templates is not
//found. Better fail now than fail later, or display rubbish.
func renderTemplate(w http.ResponseWriter, tmpl string, q interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func checks() (REDISADDR, MONGOSERVER, MONGODB string, Public []byte, Private []byte, FBURL, FBClientID, FBClientSecret, RootURL, AWSBucket string) {
	REDISADDR = os.Getenv("REDISCLOUD_URL")
	if REDISADDR == "" {
		log.Println("No mongo server address set, resulting to default address")
		REDISADDR = "localhost:6379"
	}
	log.Println("REDISADDR is ", REDISADDR)

	MONGOSERVER = os.Getenv("MONGOLAB_URI")
	if MONGOSERVER == "" {
		log.Println("No mongo server address set, resulting to default address")
		MONGOSERVER = "localhost"
	}
	log.Println("MONGOSERVER is ", MONGOSERVER)

	MONGODB = os.Getenv("MONGODB")
	if MONGODB == "" {
		log.Println("No Mongo database name set, resulting to default")
		MONGODB = "test"
	}
	log.Println("MONGODB is ", MONGODB)

	AWSBucket = os.Getenv("AWSBucket")
	if AWSBucket == "" {
		log.Println("No AWSBucket set, resulting to default")
		AWSBucket = "kakutest"
	}
	log.Println("AWSBucket is ", AWSBucket)

	Public, err := ioutil.ReadFile("app.rsa.pub")
	if err != nil {
		log.Fatal("Error reading public key")
		return
	}

	Private, err = ioutil.ReadFile("app.rsa")
	if err != nil {
		log.Fatal("Error reading private key")
		return
	}

	FBClientID = os.Getenv("FBClientID")
	FBClientSecret = os.Getenv("FBClientSecret")
	RootURL = os.Getenv("RootURL")
	if RootURL == "" {
		RootURL = "http://localhost:8080"
	}
	fbConfig := &oauth2.Config{
		// ClientId: FBAppID(string), ClientSecret : FBSecret(string)
		// Example - ClientId: "1234567890", ClientSecret: "red2drdff6e2321e51aedcc94e19c76ee"

		ClientID:     FBClientID, // change this to yours
		ClientSecret: FBClientSecret,
		RedirectURL:  RootURL + "/FBLogin", // change this to your webserver adddress
		Scopes:       []string{"email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.facebook.com/dialog/oauth",
			TokenURL: "https://graph.facebook.com/oauth/access_token",
		},
	}
	FBURL = fbConfig.AuthCodeURL("")

	if FBClientID == "" {
		FBURL = RootURL + "/offlineauth"
	}
	return
}


func serveSingle(pattern string, filename string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	})
}



func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {

	REDISADDR, MONGOSERVER, MONGODB, Public, Private, FBURL, FBClientID, FBClientSecret, RootURL, AWSBucket := checks()
	log.Println(REDISADDR)
	session, err := mgo.Dial(MONGOSERVER)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	database := session.DB(MONGODB)

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
	}
	s := s3.New(auth, aws.USWest2)
	s3bucket := s.Bucket(AWSBucket)

	rediscli := redis.NewClient(&redis.Options{
		Addr:    REDISADDR,
		Network: "tcp",
	})
	pong, err := rediscli.Ping().Result()
	log.Println(pong, err)
	if err != nil {
		panic(err)
	}

	appC := appContext{
		db:         database,
		verifyKey:  []byte(Public),
		signKey:    []byte(Private),
		token:      "AccessToken",
		login:      FBURL,
		fbclientid: FBClientID,
		fbsecret:   FBClientSecret,
		domain:     RootURL,
		bucket:     s3bucket,
		redis:      rediscli,
	}

	//appC.xmain()
	cH := alice.New(context.ClearHandler, loggingHandler)

	//serve assets
	fs := http.FileServer(http.Dir("templates/assets/"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	dist := http.FileServer(http.Dir("templates/admin/dist/"))
	http.Handle("/dist/", http.StripPrefix("/dist/", dist))

	bs := http.FileServer(http.Dir("templates/admin/bootstrap/"))
	http.Handle("/bootstrap/", http.StripPrefix("/bootstrap/", bs))

	pn := http.FileServer(http.Dir("templates/admin/plugins/"))
	http.Handle("/plugins/", http.StripPrefix("/plugins/", pn))

	//admin routes
	http.Handle("/admin", cH.Append(appC.adminAuthHandler).ThenFunc(appC.adminHandler))
	http.Handle("/admin/places/new", cH.Append(appC.adminAuthHandler).ThenFunc(appC.newplaceHandler))
	http.Handle("/admin/places/all", cH.Append(appC.adminAuthHandler).ThenFunc(appC.allPlaces))
	http.Handle("/admin/places/ua", cH.Append(appC.adminAuthHandler).ThenFunc(appC.unapprovedPlaces))
	http.Handle("/admin/places/featured", cH.Append(appC.adminAuthHandler).ThenFunc(appC.featuredPlaces))
	http.Handle("/admin/places/search", cH.Append(appC.adminAuthHandler).ThenFunc(appC.backSearchPlace))

	http.Handle("/admin/places/edit/", cH.Append(appC.adminAuthHandler).ThenFunc(appC.adminEditPlace))

	//json handlers
	http.Handle("/api/web/places/feature", cH.Append(appC.adminAuthHandler).ThenFunc(appC.featurePlace))
	http.Handle("/api/web/places/approve", cH.Append(appC.adminAuthHandler).ThenFunc(appC.approvePlace))
	http.Handle("/api/web/places", cH.ThenFunc(appC.SearchJSON))
	http.Handle("/api/web/autocomplete/places", cH.ThenFunc(appC.AutoCompletePlaces))
	http.Handle("/api/web/autocomplete/services", cH.ThenFunc(appC.AutoCompleteServices))

	http.Handle("/FBLogin", cH.ThenFunc(appC.FBLogin))
	http.Handle("/offlineauth", cH.ThenFunc(appC.offlineLogin))
	http.Handle("/logout", cH.ThenFunc(appC.logout))
	http.Handle("/loginform", cH.ThenFunc(appC.loginForm))
	http.Handle("/xxxxx", cH.ThenFunc(appC.xxxxx))

	http.Handle("/place/", cH.Append(appC.frontAuthHandler).ThenFunc(appC.singlePlace))
	http.Handle("/search", cH.Append(appC.frontAuthHandler).ThenFunc(appC.searchResultPage))
	http.Handle("/", cH.Append(appC.frontAuthHandler).ThenFunc(appC.indexHandler))

  serveSingle("/favicon.ico", "./favicon.ico")

	PORT := os.Getenv("PORT")
	if PORT == "" {
		log.Println("No Global port has been defined, using default")

		PORT = "8080"

	}

	log.Println("serving on " + appC.domain)
	log.Fatal(http.ListenAndServe(":"+PORT, http.DefaultServeMux))

}
