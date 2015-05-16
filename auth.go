package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antonholmquist/jason"
	"github.com/dgrijalva/jwt-go"
)

//AccessToken is where the facebook authentication data would be stored
type AccessToken struct {
	Token  string
	Expiry int64
}

//Handlers
func (c *appContext) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:       c.token,
		Value:      "deleted",
		Path:       "/",
		RawExpires: "0",
		//Expires:    time.Now().AddDate(0, -1, 0),
	})

	http.Redirect(w, r, "/", http.StatusFound)

}

func (c *appContext) offlineLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("Offline Login")
	u := User{
		Name:  "Anthony Alaribe",
		PID:   "1234567890",
		Email: "anthonyalaribe@gmail.com",
		Image: "http://placehold.it/100x100",
	}

	user, err := c.Authenticate(&u, "facebook")

	if err != nil {
		log.Println(err)
	}
log.Println(user)
	// create a signer for rsa 256
	t := jwt.New(jwt.GetSigningMethod("RS256"))

	// set our claims
	t.Claims["AccessToken"] = user.Permission
	t.Claims["User"] = user

	// set the expire time
	// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4
	t.Claims["exp"] = time.Now().Add(time.Minute * 10).Unix()
	tokenString, err := t.SignedString(c.signKey)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Sorry, error while Signing Token!")
		log.Printf("Token Signing error: %v\n", err)
		return
	}

	log.Println(c.token)
	log.Println(tokenString)

	// i know using cookies to store the token isn't really helpfull for cross domain api usage
	// but it's just an example and i did not want to involve javascript
	http.SetCookie(w, &http.Cookie{
		Name:       c.token,
		Value:      tokenString,
		Path:       "/",
		RawExpires: "0",
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (c *appContext) FBLogin(w http.ResponseWriter, r *http.Request) {
	// grab the code fragment

	code := r.FormValue("code")

	RedirectURL := c.domain + "/FBLogin"

	accessToken := GetAccessToken(c.fbclientid, code, c.fbsecret, RedirectURL)

	response, err := http.Get("https://graph.facebook.com/me?access_token=" + accessToken.Token)

	// handle err. You need to change this into something more robust
	// such as redirect back to home page with error message
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	str := readHTTPBody(response)
	log.Println(str)
	us, err := jason.NewObjectFromBytes([]byte(str))
	if err != nil {
		log.Println(err)
	}

	id, err := us.GetString("id")
	if err != nil {
		log.Println(err)
	}

	email, err := us.GetString("email")
	if err != nil {
		log.Println(err)
	}

	name, err := us.GetString("name")
	if err != nil {
		log.Println(err)
	}

	img := "https://graph.facebook.com/" + id + "/picture?width=180&height=180"
	log.Println("It got this far; the ID is")
	log.Println(id)

	u := User{
		Name:  name,
		PID:   id,
		Email: email,
		Image: img,
	}

	user, err := c.Authenticate(&u, "facebook")
  log.Println(user)
	if err != nil {
		log.Println(err)
	}

	// create a signer for rsa 256
	t := jwt.New(jwt.GetSigningMethod("RS256"))

	// set our claims
	t.Claims["AccessToken"] = user.Permission
	t.Claims["User"] = user

	// set the expire time
	// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4

	t.Claims["exp"] = time.Now().Add(time.Hour * 99999).Unix()
	tokenString, err := t.SignedString(c.signKey)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Sorry, error while Signing Token!")
		log.Printf("Token Signing error: %v\n", err)
		return
	}

	// i know using cookies to store the token isn't really helpfull for cross domain api usage
	// but it's just an example and i did not want to involve javascript
	http.SetCookie(w, &http.Cookie{
		Name:       c.token,
		Value:      tokenString,
		Path:       "/",
		RawExpires: "0",
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func readHTTPBody(response *http.Response) string {

	log.Println("Reading body")

	bodyBuffer := make([]byte, 5000)
	var str string

	count, err := response.Body.Read(bodyBuffer)

	for ; count > 0; count, err = response.Body.Read(bodyBuffer) {

		if err != nil {

		}

		str += string(bodyBuffer[:count])
	}

	return str

}

// GetAccessToken Converts a code to an Auth_Token
func GetAccessToken(clientID string, code string, secret string, callbackURI string) AccessToken {
	log.Println("GetAccessToken")
	//https://graph.facebook.com/oauth/access_token?client_id=YOUR_APP_ID&redirect_uri=YOUR_REDIRECT_URI&client_secret=YOUR_APP_SECRET&code=CODE_GENERATED_BY_FACEBOOK
	response, err := http.Get("https://graph.facebook.com/oauth/access_token?client_id=" +
		clientID + "&redirect_uri=" + callbackURI +
		"&client_secret=" + secret + "&code=" + code)

	if err == nil {

		auth := readHTTPBody(response)

		log.Println(auth)
		var token AccessToken

		tokenArr := strings.Split(auth, "&")
		log.Println(tokenArr)

		token.Token = strings.Split(tokenArr[0], "=")[1]
		expireInt, err := strconv.Atoi(strings.Split(tokenArr[1], "=")[1])

		if err == nil {
			token.Expiry = int64(expireInt)
		}

		return token
	}

	var token AccessToken

	return token
}
