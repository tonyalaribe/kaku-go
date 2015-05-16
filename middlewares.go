package main

import (
	"log"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
		"github.com/mitchellh/mapstructure"
)

//Middlewares

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				//WriteError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (ac *appContext) adminAuthHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// check if we have a cookie with out tokenName

		tokenCookie, err := r.Cookie(ac.token)
		log.Println(err)
		//log.Println(ac.token)
		//log.Println(tokenCookie)

		switch {
		case err == http.ErrNoCookie:
			http.Redirect(w, r, "/loginform", http.StatusFound)

		case err != nil:
			http.Redirect(w, r, "/loginform", http.StatusFound)
		}
		// validate the token
		token, err := jwt.Parse(tokenCookie.Value, func(token *jwt.Token) (interface{}, error) {
			// since we only use the one private key to sign
			// the tokens,
			// we also only use its public counter
			// part to verify
			return ac.verifyKey, nil
		})

		// branch out into the possible error from signing
		switch err.(type) {

		case nil: // no error

			if !token.Valid { // but may still be invalid
				http.Redirect(w, r, "/loginform", http.StatusFound)
			}

			log.Println("Someone accessed resricted area! Token:%+v\n", token)
			//w.Header().Set("Content-Type", "text/html")
			//w.WriteHeader(http.StatusOK)
			//fmt.Fprintln(w, "restricted Area")
      u := token.Claims["User"]
      var user User
    	err := mapstructure.Decode(u, &user)
    	if err != nil {
    		log.Println(err)
    	}
    	
    	log.Println(user)
    	log.Println(user.Permission)
    	log.Println("*************************************************************************")
      if user.Permission == "admin"{
  			context.Set(r, "User", u)
  			log.Println(u)
  			next.ServeHTTP(w, r)
      }else{
        http.Redirect(w, r, "/loginform", http.StatusFound)
      }
		case *jwt.ValidationError: // something was wrong during the validation
			vErr := err.(*jwt.ValidationError)

			switch vErr.Errors {
			case jwt.ValidationErrorExpired:
				http.Redirect(w, r, "/loginform", http.StatusFound)

			default:
				http.Redirect(w, r, "/loginform", http.StatusFound)
			}

		default: // something else went wrong
			http.Redirect(w, r, "/loginform", http.StatusFound)
		}

	}
	return http.HandlerFunc(fn)

}



func (ac *appContext) frontAuthHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// check if we have a cookie with out tokenName

		tokenCookie, err := r.Cookie(ac.token)
		if err !=nil{
		  log.Println(err)
		}
		//log.Println(ac.token)
		//log.Println(tokenCookie)

		switch {
		case err == http.ErrNoCookie:
			//w.WriteHeader(http.StatusUnauthorized)
			//fmt.Fprintln(w, "No Token, no fun!")
			next.ServeHTTP(w, r)

		case err != nil:
			//w.WriteHeader(http.StatusInternalServerError)
			//fmt.Fprintln(w, "Error while Parsing cookie!")
			log.Printf("Cookie parse error: %v\n", err)
			next.ServeHTTP(w, r)
		}
		// validate the token
		token, err := jwt.Parse(tokenCookie.Value, func(token *jwt.Token) (interface{}, error) {
			// since we only use the one private key to sign
			// the tokens,
			// we also only use its public counter
			// part to verify
			return ac.verifyKey, nil
		})

		// branch out into the possible error from signing
		switch err.(type) {

		case nil: // no error

			if !token.Valid { // but may still be invalid
				//w.WriteHeader(http.StatusUnauthorized)
				//fmt.Fprintln(w, "WHAT? Invalid Token? F*** off!")
				log.Println("Invalid Token.... Hack attempt?")
			}

			//log.Println("Someone accessed resricted area! Token:%+v\n", token)
			//w.Header().Set("Content-Type", "text/html")
			//w.WriteHeader(http.StatusOK)
			//fmt.Fprintln(w, "restricted Area")

			context.Set(r, "User", token.Claims["User"])
			log.Println(token.Claims["User"])
			next.ServeHTTP(w, r)

		case *jwt.ValidationError: // something was wrong during the validation
			vErr := err.(*jwt.ValidationError)

			switch vErr.Errors {
			case jwt.ValidationErrorExpired:
				//w.WriteHeader(http.StatusUnauthorized)
				//fmt.Fprintln(w, "Token Expired, get a new one.")
				next.ServeHTTP(w, r)

			default:
				//w.WriteHeader(http.StatusInternalServerError)
				//fmt.Fprintln(w, "Error while Parsing Token!")
				log.Printf("ValidationError error: %+v\n", vErr.Errors)
				next.ServeHTTP(w, r)
			}

		default: // something else went wrong
			//w.WriteHeader(http.StatusInternalServerError)
			//fmt.Fprintln(w, "Error while Parsing Token!")
			log.Printf("Token parse error: %v\n", err)
			next.ServeHTTP(w, r)
		}

	}
	return http.HandlerFunc(fn)

}
