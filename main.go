package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/jaredfolkins/badactor"
	"github.com/julienschmidt/httprouter"
)

var st *badactor.Studio

type MyAction struct{}

func (ma *MyAction) WhenJailed(a *badactor.Actor, r *badactor.Rule) error {
	return nil
}

func (ma *MyAction) WhenTimeServed(a *badactor.Actor, r *badactor.Rule) error {
	return nil
}

func main() {

	//runtime.GOMAXPROCS(4)

	// studio capacity
	var sc int32
	// director capacity
	var dc int32

	sc = 1024
	dc = 1024

	// init new Studio
	st = badactor.NewStudio(sc)

	// define and add the rule to the stack
	ru := &badactor.Rule{
		Name:        "Login",
		Message:     "You have failed to login too many times",
		StrikeLimit: 10,
		ExpireBase:  time.Second * 1,
		Sentence:    time.Second * 10,
		Action:      &MyAction{},
	}
	st.AddRule(ru)

	err := st.CreateDirectors(dc)
	if err != nil {
		log.Fatal(err)
	}

	// Start the reaper
	st.StartReaper()

	// router
	router := httprouter.New()
	router.POST("/login", LoginHandler)

	// middleware
	n := negroni.Classic()
	n.Use(NewBadActorMiddleware())
	n.UseHandler(router)
	n.Run(":9999")

}

//
// HANDLER
//

// this is a niave login function for example purposes
func LoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	var err error

	un := r.FormValue("username")
	pw := r.FormValue("password")

	// snag the IP for use as the actor's name
	an, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(err)
	}

	// mock authentication
	if un == "example_user" && pw == "example_pass" {
		http.Redirect(w, r, "", http.StatusOK)
		return
	}

	// auth fails, increment infraction
	err = st.Infraction(an, "Login")
	if err != nil {
		log.Printf("[%v] has err %v", an, err)
	}

	// auth fails, increment infraction
	i, err := st.Strikes(an, "Login")
	log.Printf("[%v] has %v Strikes %v", an, i, err)

	http.Redirect(w, r, "", http.StatusUnauthorized)
	return
}

//
// MIDDLEWARE
//
type BadActorMiddleware struct {
	negroni.Handler
}

func NewBadActorMiddleware() *BadActorMiddleware {
	return &BadActorMiddleware{}
}

func (bam *BadActorMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	// snag the IP for use as the actor's name
	an, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(err)
	}

	// if the Actor is jailed, send them StatusUnauthorized
	if st.IsJailed(an) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// call the next middleware in the chain
	next(w, r)
}
