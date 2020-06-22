/**
 * Copyright 2018 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * @fileoverview Service worker for Firebase Auth test app application. The
 * service worker caches all content and only serves cached content in offline
 * mode.
 */

import firebase from "firebase/app";
import "firebase/auth";
import config from "./firebase-config.js";

// Initialize the Firebase app in the web worker.
firebase.initializeApp(config);

/**
 * Returns a promise that resolves with an ID token if available.
 * @return {!Promise<?string>} The promise that resolves with an ID token if
 *     available. Otherwise, the promise resolves with null.
 */
const getIdToken = () => {
  // eslint-disable-next-line no-unused-vars
  return new Promise((resolve, reject) => {
    const unsubscribe = firebase.auth().onAuthStateChanged(user => {
      unsubscribe();
      if (user) {
        user.getIdToken().then(
          idToken => {
            resolve(idToken);
          },
          error => {
            console.log("55: ", error);
            resolve(null);
          }
        );
      } else {
        resolve(null);
      }
    });
  }).catch(error => {
    console.log("63: ", error);
  });
};

/**
 * @param {string} url The URL whose origin is to be returned.
 * @return {string} The origin corresponding to given URL.
 */
const getOriginFromUrl = url => {
  // https://stackoverflow.com/questions/1420881/how-to-extract-base-url-from-a-string-in-javascript
  const pathArray = url.split("/");
  const protocol = pathArray[0];
  const host = pathArray[2];
  return protocol + "//" + host;
};

// As this is a test app, let's only return cached data when offline.
self.addEventListener("fetch", event => {
  const requestProcessor = idToken => {
    let req = event.request.clone();
    // For same origin https requests, append idToken to header.
    if (
      self.location.origin == getOriginFromUrl(event.request.url) &&
      (self.location.protocol == "https:" ||
        self.location.hostname == "localhost") &&
      idToken
    ) {
      // Clone headers as request headers are immutable.
      const headers = new Headers(req.headers);
      // Add ID token to header.
      headers.append("Authorization", "Bearer " + idToken);
      try {
        req = new Request(req, {
          headers: headers
        });
      } catch (e) {
        // This will fail for CORS requests. We just continue with the
        // fetch caching logic below and do not pass the ID token.
      }
    }
    return fetch(req);
  };
  // Fetch the resource after checking for the ID token.
  // This can also be integrated with existing logic to serve cached files
  // in offline mode.
  event.respondWith(getIdToken().then(requestProcessor, requestProcessor));
});

self.addEventListener("activate", function(event) {
  event.waitUntil(self.clients.claim());
});
