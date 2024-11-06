package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func handleSentinelLogin(w http.ResponseWriter, r *http.Request) {
	urlStr := sentinelConfig.AuthCodeURL(oauthStateString) // Generate the Sentinel consent page URL

	baseAuthURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("Error parsing auth URL: %v", err)
	}

	// Add the PKCE parameters (code_challenge and code_challenge_method)
	query := baseAuthURL.Query()
	query.Set("code_challenge", generateCodeChallenge(codeVerifier))
	query.Set("code_challenge_method", "S256")
	baseAuthURL.RawQuery = query.Encode()

	urlStr = baseAuthURL.String()
	http.Redirect(w, r, urlStr, http.StatusTemporaryRedirect) // Redirect user to Sentinel consent page
}

func handleSentinelCallback(w http.ResponseWriter, r *http.Request) {
	// Validate the state parameter to prevent CSRF attacks
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Printf("invalid OAuth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Get the authorization code from the query string
	code := r.FormValue("code")

	// Exchange the authorization code for an access token
	token, err := sentinelConfig.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		log.Printf("failed to exchange token: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Print the access token details
	fmt.Fprintf(w, "Sentinel Access Token: %s\n", token.AccessToken)
	idToken := token.Extra("id_token")
	fmt.Fprintf(w, "Sentinel ID Token: %s\n", idToken.(string)) // Get ID token (JWT token)
	fmt.Fprintf(w, "Token Type: %s\n", token.TokenType)
	fmt.Fprintf(w, "Expiry: %s\n", token.Expiry)

	// Validate the ID token
	parsedToken, err := validateToken(sentinelJwksURL, token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate ID token: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract claims (including user ID)
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {

		fmt.Fprintf(w, "Sentinel Token Claims:")
		for key, value := range claims {
			fmt.Fprintf(w, "%s: %v\n", key, value)
		}
	} else {
		http.Error(w, "Invalid token", http.StatusInternalServerError)
	}
}
