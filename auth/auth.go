package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/pivot"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// User :
type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Account  string `json:"account"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
}

// Credentials :
type Credentials struct {
	Account  string `json:"account"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token"`
	Role     string `json:"role"`
}

// TokenAuth :
type TokenAuth struct {
	tokenStore          *JwtStore
	store               katamari.Database
	getter              TokenGetter
	UnauthorizedHandler http.HandlerFunc
	client              *http.Client
}

// TokenGetter :
type TokenGetter interface {
	GetTokenFromRequest(req *http.Request) string
}

// Token :
type Token interface {
	IsExpired() bool
	fmt.Stringer
	ClaimGetter
}

// ClaimSetter :
type ClaimSetter interface {
	SetClaim(string, interface{}) ClaimSetter
}

// ClaimGetter :
type ClaimGetter interface {
	Claims(string) interface{}
}

// BearerGetter :
type BearerGetter struct {
	Header string
}

var (
	userRegexp  = regexp.MustCompile("^[a-zA-Z0-9_]{2,15}$")
	emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	phoneRegexp = regexp.MustCompile("^[0-9_-]{6,15}$")
	roles       = map[string]string{"root": "root"}
)

// DefaultUnauthorizedHandler :
func DefaultUnauthorizedHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "unauthorized")
}

// GetTokenFromRequest :
func (b *BearerGetter) GetTokenFromRequest(req *http.Request) string {
	// log.Println("header:", req.Header)
	authStr := req.Header.Get(b.Header)
	if !strings.HasPrefix(authStr, "Bearer ") {
		return ""
	}

	return authStr[7:]
}

// NewHeaderBearerTokenGetter :
func NewHeaderBearerTokenGetter(header string) *BearerGetter {
	return &BearerGetter{
		Header: header,
	}
}

// New :
//
// Returns a TokenAuth object implemting Handler interface
//
// if a handler is given it proxies the request to the handler
//
// if a unauthorizedHandler is provided, unauthorized requests
//
// will be handled by this HandlerFunc, otherwise a default
//
// unauthorized handler is used.
//
// store is the TokenStore that stores and verify the tokens
//
func New(tokenStore *JwtStore, store katamari.Database) *TokenAuth {
	t := &TokenAuth{
		tokenStore: tokenStore,
		store:      store,
	}
	t.getter = NewHeaderBearerTokenGetter("Authorization")
	t.UnauthorizedHandler = DefaultUnauthorizedHandler
	return t
}

// Verify : wrap a HandlerFunc to be authenticated
func (t *TokenAuth) Verify(req *http.Request) bool {
	_, err := t.Authenticate(req)
	if err != nil {
		return false
	}
	// context.Set(req, "token", token)
	return true
}

// Authenticate :
func (t *TokenAuth) Authenticate(r *http.Request) (Token, error) {
	strToken := t.getter.GetTokenFromRequest(r)
	if strToken == "" {
		return nil, errors.New("token required")
	}
	token, err := t.tokenStore.CheckToken(strToken)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Audit : get websocket token, return token claims
func (t *TokenAuth) Audit(r *http.Request) (string, string, error) {
	// get the header from a websocket connection
	// https://stackoverflow.com/questions/22383089/is-it-possible-to-use-bearer-authentication-for-websocket-upgrade-requests
	if r.Header.Get("Upgrade") == "websocket" && r.Header.Get("Sec-WebSocket-Protocol") != "" {
		r.Header.Add("Authorization", "Bearer "+strings.Replace(r.Header.Get("Sec-WebSocket-Protocol"), "bearer, ", "", 1))
	}
	token, err := t.Authenticate(r)
	if err != nil {
		return "", "", err
	}
	role := token.Claims("role").(string)
	account := token.Claims("iss").(string)
	return role, account, nil
}

// Authorize method
func (t *TokenAuth) getUser(account string) (User, error) {
	var user User
	raw, err := t.store.Get("users/" + account)
	if err != nil {
		return user, err
	}
	var obj objects.Object
	err = json.Unmarshal(raw, &obj)
	if err != nil {
		return user, err
	}
	err = json.Unmarshal([]byte(obj.Data), &user)
	if err != nil {
		return user, err
	}
	return user, nil
}

func (t *TokenAuth) getUsers() ([]User, error) {
	var users []User
	raw, err := t.store.Get("users/*")
	if err != nil {
		return nil, err
	}
	var objects []objects.Object
	err = json.Unmarshal(raw, &objects)
	if err != nil {
		return nil, err
	}
	for _, object := range objects {
		var user User
		err = json.Unmarshal([]byte(object.Data), &user)
		if err == nil {
			user.Password = ""
			users = append(users, user)
		}
	}
	return users, nil
}

func getCredentials(r *http.Request) (Credentials, error) {
	dec := json.NewDecoder(r.Body)
	var credentials Credentials
	err := dec.Decode(&credentials)
	if err != nil {
		return credentials, err
	}

	return credentials, nil
}

func (t *TokenAuth) checkCredentials(credentials Credentials) (User, error) {
	user, err := t.getUser(credentials.Account)
	if err != nil {
		return user, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		return user, errors.New("wrong password")
	}

	return user, nil
}

// Profile returns to the client the correspondent user profile for the token provided
func (t *TokenAuth) Profile(pivotIP string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if pivotIP != "" {
			pivot.Synchronize(t.client, t.store, pivotIP, []string{"users/*"})
		}
		token, err := t.Authenticate(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("this request is not authorized"))
			return
		}
		switch r.Method {
		case "GET":
			user, err := t.getUser(token.Claims("iss").(string))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "bad token, couldnt find the issuer profile")
				return
			}
			user.Password = ""
			w.WriteHeader(http.StatusOK)
			enc := json.NewEncoder(w)
			enc.Encode(&user)
			return
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Method not suported")
			return
		}
	}
}

// Authorize will claim a token on POST and refresh the claim on PUT
func (t *TokenAuth) Authorize(pivotIP string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if pivotIP != "" {
			pivot.Synchronize(t.client, t.store, pivotIP, []string{"users/*"})
		}
		credentials, err := getCredentials(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
		user, err := t.getUser(credentials.Account)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
		switch r.Method {
		case "POST":
			_, err = t.checkCredentials(credentials)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, err.Error())
				return
			}
			break
		case "PUT":
			if credentials.Token != "" {
				oldToken, err := t.tokenStore.CheckToken(credentials.Token)
				if err == nil {
					w.WriteHeader(http.StatusNotModified)
					fmt.Fprint(w, errors.New("token not expired"))
					return
				}

				if oldToken.Claims("iss").(string) != credentials.Account {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, errors.New("token doesn't match the account"))
					return
				}

				if err.Error() != "Token expired" {
					w.WriteHeader(http.StatusNotModified)
					fmt.Fprint(w, err)
					return
				}
			} else {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, errors.New("empty token"))
				return
			}
			break
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Method not suported")
			return
		}
		newToken := t.tokenStore.NewToken()
		newToken.SetClaim("iss", credentials.Account)
		newToken.SetClaim("role", user.Role)
		credentials.Password = ""
		credentials.Role = user.Role
		credentials.Token = newToken.String()
		w.Header().Add("content-type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(&credentials)
	}
}

// Register will create a new user
func (t *TokenAuth) Register(w http.ResponseWriter, r *http.Request) {
	var user User
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&user)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	if user.Account == "" || user.Name == "" || user.Password == "" || user.Email == "" || user.Phone == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("new user data incomplete"))
		return
	}

	if !userRegexp.MatchString(user.Account) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account cannot contain special characters, only numbers or lowercase letters and character count must be between 2 and 15"))
		return
	}

	if len(user.Password) < 3 || len(user.Password) > 88 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("password character count must be between 2 and 88"))
		return
	}

	if !userRegexp.MatchString(user.Phone) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("phone cannot contain special characters othen than '-' and character count must be between 6 and 15"))
		return
	}

	if !emailRegexp.MatchString(user.Email) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("invalid email address"))
		return
	}

	_, err = t.getUser(user.Account)

	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account name taken"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	user.Password = string(hash)
	user.Role = "user"
	role, otherRole := roles[user.Account]
	if otherRole {
		user.Role = role
	}
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(user)
	_, err = t.store.Set("users/"+user.Account, string(dataBytes.Bytes()))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	newToken := t.tokenStore.NewToken()
	newToken.SetClaim("iss", user.Account)
	newToken.SetClaim("role", user.Role)
	credentials := Credentials{
		Account: user.Account,
		Token:   newToken.String(),
		Role:    user.Role,
	}
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(&credentials)
}

// NewPassword is for updating account password
func (t *TokenAuth) NewPassword(w http.ResponseWriter, r *http.Request) {
	token, err := t.Authenticate(r)
	authorized := (err == nil)
	role := "user"
	if authorized {
		role = token.Claims("role").(string)
	}

	// root authorization
	if role != "root" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "Method not suported for your role")
		return
	}
	account := mux.Vars(r)["account"]

	user, err := t.getUser(account)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, err.Error())
		return
	}
	switch r.Method {
	case "PUT":
		dec := json.NewDecoder(r.Body)
		var userData User
		err := dec.Decode(&userData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, errors.New("Invalid user data"))
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.MinCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
		user.Password = string(hash)

		dataBytes := new(bytes.Buffer)
		json.NewEncoder(dataBytes).Encode(user)
		_, err = t.store.Set("users/"+user.Account, string(dataBytes.Bytes()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		user.Password = ""
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(&user)
		break

	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Method not suported")
		return
	}
}

// Available will check if an account is taken
func (t *TokenAuth) Available(pivotIP string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if pivotIP != "" {
			pivot.Synchronize(t.client, t.store, pivotIP, []string{"users/*"})
		}
		account := r.FormValue("account")
		_, err := t.getUser(account)

		if err == nil {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "account taken")
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "account available")
	}
}

// Create will create a new user (root only access)
func (t *TokenAuth) Create(w http.ResponseWriter, r *http.Request) {
	token, err := t.Authenticate(r)
	authorized := (err == nil)
	role := "user"
	if authorized {
		role = token.Claims("role").(string)
	}

	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Method not suported for your role")
		return
	}

	// root authorization
	if role != "root" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "Method not suported for your role")
		return
	}
	var user User
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&user)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	if user.Account == "" || user.Name == "" || user.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("new user data incomplete"))
		return
	}

	if !userRegexp.MatchString(user.Account) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account cannot contain special characters, only numbers or lowercase letters and character count must be between 2 and 15"))
		return
	}

	if len(user.Password) < 3 || len(user.Password) > 88 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("password character count must be between 2 and 88"))
		return
	}

	_, err = t.getUser(user.Account)

	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account name taken"))
		return
	}
}

// Users will send the user list to a root user
func (t *TokenAuth) Users(pivotIP string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := t.Authenticate(r)
		authorized := (err == nil)
		role := "user"
		if authorized {
			role = token.Claims("role").(string)
		}

		// root authorization
		if !authorized || role != "root" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "Method not suported for your role")
			return
		}

		if pivotIP != "" {
			pivot.Synchronize(t.client, t.store, pivotIP, []string{"users/*"})
		}

		users, err := t.getUsers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(&users)
	}
}

// User will send the user list to a root user
func (t *TokenAuth) User(w http.ResponseWriter, r *http.Request) {
	token, err := t.Authenticate(r)
	authorized := (err == nil)
	role := "user"
	if authorized {
		role = token.Claims("role").(string)
	}

	// root authorization
	if !authorized || role != "root" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "Method not suported for your role")
		return
	}
	account := mux.Vars(r)["account"]

	user, err := t.getUser(account)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, err.Error())
		return
	}
	switch r.Method {
	case "GET":
		user.Password = ""
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(&user)
		break
	case "DELETE":
		err := t.store.Del("users/" + user.Account)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "deleted "+account)
		break
	case "POST":
		dec := json.NewDecoder(r.Body)
		var userData User
		err := dec.Decode(&userData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, errors.New("Invalid user data"))
			return
		}
		if userData.Email != "" {
			user.Email = userData.Email
		}
		if userData.Name != "" {
			user.Name = userData.Name
		}
		if userData.Phone != "" {
			user.Phone = userData.Phone
		}
		if userData.Role != "" {
			user.Role = userData.Role
		}
		dataBytes := new(bytes.Buffer)
		json.NewEncoder(dataBytes).Encode(user)
		_, err = t.store.Set("users/"+user.Account, string(dataBytes.Bytes()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		user.Password = ""
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(&user)
		break
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Method not suported")
		return
	}
}

// Router handle for auth enpoints
func (t *TokenAuth) Router(server *katamari.Server, pivotIP string) {
	server.Router.HandleFunc("/authorize", t.Authorize(pivotIP))
	server.Router.HandleFunc("/profile", t.Profile(pivotIP))
	server.Router.HandleFunc("/users", t.Users(pivotIP)).Methods("GET")
	server.Router.HandleFunc("/user/{account:[a-zA-Z\\d]+}", t.User).Methods("GET", "POST", "DELETE")
	server.Router.HandleFunc("/password/{account:[a-zA-Z\\d]+}", t.NewPassword).Methods("PUT")
	server.Router.HandleFunc("/register", t.Register).Methods("POST")
	server.Router.HandleFunc("/create", t.Create).Methods("POST")
	server.Router.HandleFunc("/available", t.Available(pivotIP)).Queries("account", "{[a-zA-Z\\d]}").Methods("GET")

	t.client = server.Client
	pivot.Router(server.Router, t.store, server.Client, pivotIP, []string{"users/*"})
}
