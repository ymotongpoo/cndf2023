// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import http from 'k6/http';
import { sleep } from 'k6';

const SERVICE_ENDPOINT="https://shakesapp-loiwv2t7ea-de.a.run.app"
const words = ["hello", "love", "life", "people", "cloud", "sun", "rainbow", "beauty"]

export const options = {
    vus: 10,
    duration: '600s',
}

export default function () {
    url = genRequestURL()
    http.get(url);
    sleep(5);
}

const genRequestURL = () => {
    word = words[Math.floor(Math.random() * words.length)];
    return SERVICE_ENDPOINT + `?q=${param}`;
}