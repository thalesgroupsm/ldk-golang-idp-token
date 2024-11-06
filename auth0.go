package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
)

// Login handler: Redirects user to Auth0's authorization endpoint
func handleAuth0Login(w http.ResponseWriter, r *http.Request) {
	urlStr := auth0Config.AuthCodeURL(oauthStateString) // Generate the Auth0 consent page URL
	baseAuthURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("Error parsing auth URL: %v", err)
	}

	query := baseAuthURL.Query()
	query.Set("audience", auth0Audience)
	baseAuthURL.RawQuery = query.Encode()

	// Redirect user to Auth0 consent page
	urlStr = baseAuthURL.String()
	http.Redirect(w, r, urlStr, http.StatusTemporaryRedirect)
}

// Callback handler: Receives the authorization code and exchanges it for an access token
func handleAuth0Callback(w http.ResponseWriter, r *http.Request) {
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
	token, err := auth0Config.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("failed to exchange token: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Print the access token details
	accessToken := token.Extra("access_token")
	fmt.Fprintf(w, "Auth0 Access Token: %s\n", accessToken)
	idToken := token.Extra("id_token")
	fmt.Fprintf(w, "Auth0 ID Token: %s\n", idToken.(string)) // Get ID token (JWT token)
	fmt.Fprintf(w, "Token Type: %s\n", token.TokenType)
	fmt.Fprintf(w, "Expiry: %s\n", token.Expiry)

	// Validate the ID token
	parsedIdToken, err := validateToken(auth0JwksURL, idToken.(string))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate ID token: %v", err), http.StatusInternalServerError)
		return
	}

	if claims, ok := parsedIdToken.Claims.(jwt.MapClaims); ok && parsedIdToken.Valid {
		fmt.Fprintf(w, "Auth0 Id Token Claims:\n")
		for key, value := range claims {
			fmt.Fprintf(w, "%s: %v\n", key, value)
		}
	} else {
		http.Error(w, "Invalid ID token", http.StatusInternalServerError)
	}

	// Validate the ID token
	parsedAccessToken, err := validateToken(auth0JwksURL, token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to validate access token: %v", err), http.StatusInternalServerError)
		return
	}

	if claims, ok := parsedAccessToken.Claims.(jwt.MapClaims); ok && parsedAccessToken.Valid {
		fmt.Fprintf(w, "Access Token Claims:\n")
		for key, value := range claims {
			fmt.Fprintf(w, "%s: %v\n", key, value)
		}
	} else {
		http.Error(w, "Invalid ID token", http.StatusInternalServerError)
	}
}