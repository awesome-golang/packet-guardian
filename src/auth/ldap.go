package auth

import (
	"net/http"

	ldapc "github.com/dragonrider23/go-ldap-client"
	"github.com/dragonrider23/verbose"
	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/models"
	"gopkg.in/ldap.v2"
)

func init() {
	authFunctions["ldap"] = &ldapAuthenticator{}
}

type ldapAuthenticator struct {
	client *ldapc.LDAPClient
}

func (l *ldapAuthenticator) loginUser(r *http.Request, w http.ResponseWriter) bool {
	e := common.GetEnvironmentFromContext(r)
	// TODO: Support full LDAP servers and not just AD
	// TODO: Support multiple LDAP servers, not just one
	if l.client == nil {
		l.client = &ldapc.LDAPClient{
			Host:         e.Config.Auth.LDAP.Servers[0],
			Port:         389,
			UseSSL:       e.Config.Auth.LDAP.VerifySSLCert,
			ADDomainName: e.Config.Auth.LDAP.DomainName,
		}
	}
	defer l.client.Close()

	ok, _, err := l.client.Authenticate(r.FormValue("username"), r.FormValue("password"))
	if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
		e.Log.WithFields(verbose.Fields{
			"Err":      err,
			"Username": r.FormValue("username"),
		}).Errorf("Error authenticating with LDAP server")
	}

	if !ok {
		return false
	}

	user, err := models.GetUserByUsername(e, r.FormValue("username"))
	if err != nil {
		e.Log.WithField("Err", err).Errorf("Error getting user")
		return false
	}
	if user.IsExpired() {
		e.Log.WithField("username", user.Username).Info("Failed login by expired user")
		return false
	}

	sess := common.GetSessionFromContext(r)
	sess.Set("loggedin", true)
	sess.Set("username", user.Username)
	sess.Save(r, w)
	return true
}

func (l *ldapAuthenticator) logoutUser(r *http.Request, w http.ResponseWriter) {
	sess := common.GetSessionFromContext(r)
	if sess.GetBool("loggedin") {
		sess.Set("loggedin", false)
		sess.Set("username", "")
		sess.Save(r, w)
	}
}

func (l *ldapAuthenticator) isLoggedIn(r *http.Request) bool {
	return common.GetSessionFromContext(r).GetBool("loggedin")
}
