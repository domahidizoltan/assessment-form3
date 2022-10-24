# Form3 Take Home Exercise

name: Zoltán Domahidi  
date: October 24, 2022

*Scroll down for the original description.*

Notes:
- Adminer is added to the `docker-compose.yml` so the database could be viewed from a browser if there is no Postgres client application installed.
  - Url: http://localhost:9000/
  - System: PostgreSQL
  - Server: postgresql
  - Username: interview_accountapi_user
  - Password: 123
  - Database: interview_accountapi  
<br/>

- [This](https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts) document served as a base for implementing the requests however some changes are not documented (not even in the public Postman collection) or the fake API has some differences:
  - deleting an inexistent account without version results a Bad Request (invalid version id) and not Not Found
  - on create:
    - id is required so it is not auto-generated by the server (nil UUID is also accepted)
    - `data.attributes.base_currency` is documented as required but it's not
    - the following fields are documented as not required but they are (or they are not documented at all): `data.id`, `data.organisation_id`, `data.type`, `data.attributes`, `data.attributes.country`, `data.attributes.name`
    - in the `Create an Account` section the statements about account number and IBAN generations are not true  
<br/>

- Advanced features should not be implemented however I was thinking that a client library should expose some metrics or help somehow the caller to add some extra things around the calls. First I was thinking just simply add Prometheus metrics but I guess it's more flexible to allow the caller to use hooks and pass it's own context. `RequestEnricher` (probably not the best name for it) in `form3interview/pkg/requestenricher` tries to give a simple solution for this. 
  - As I checked it variable shadowing not works from within the hooks so it should be safe to be used. 
  - The response in `AfterHook` does not contain the response Body.  
<br/>


<br/>

---

*Original description below.*

---
<br/>
<br/>

# Form3 Take Home Exercise

Engineers at Form3 build highly available distributed systems in a microservices environment. Our take home test is designed to evaluate real world activities that are involved with this role. We recognise that this may not be as mentally challenging and may take longer to implement than some algorithmic tests that are often seen in interview exercises. Our approach however helps ensure that you will be working with a team of engineers with the necessary practical skills for the role (as well as a diverse range of technical wizardry). 

## Instructions
The goal of this exercise is to write a client library in Go to access our fake account API, which is provided as a Docker
container in the file `docker-compose.yaml` of this repository. Please refer to the
[Form3 documentation](https://www.api-docs.form3.tech/api/tutorials/getting-started/create-an-account) for information on how to interact with the API. Please note that the fake account API does not require any authorisation or authentication.

A mapping of account attributes can be found in [models.go](./models.go). Can be used as a starting point, usage of the file is not required.

If you encounter any problems running the fake account API we would encourage you to do some debugging first,
before reaching out for help.

## Submission Guidance

### Shoulds

The finished solution **should:**
- Be written in Go.
- Use the `docker-compose.yaml` of this repository.
- Be a client library suitable for use in another software project.
- Implement the `Create`, `Fetch`, and `Delete` operations on the `accounts` resource.
- Be well tested to the level you would expect in a commercial environment. Note that tests are expected to run against the provided fake account API.
- Be simple and concise.
- Have tests that run from `docker-compose up` - our reviewers will run `docker-compose up` to assess if your tests pass.

### Should Nots

The finished solution **should not:**
- Use a code generator to write the client library.
- Use (copy or otherwise) code from any third party without attribution to complete the exercise, as this will result in the test being rejected.
    - **We will fail tests that plagiarise others' work. This includes (but is not limited to) other past submissions or open-source libraries.**
- Use a library for your client (e.g: go-resty). Anything from the standard library (such as `net/http`) is allowed. Libraries to support testing or types like UUID are also fine.
- Implement client-side validation.
- Implement an authentication scheme.
- Implement support for the fields `data.attributes.private_identification`, `data.attributes.organisation_identification`
  and `data.relationships` or any other fields that are not included in the provided `models.go`, as they are omitted from the provided fake account API implementation.
- Have advanced features, however discussion of anything extra you'd expect a production client to contain would be useful in the documentation.
- Be a command line client or other type of program - the requirement is to write a client library.
- Implement the `List` operation.
> We give no credit for including any of the above in a submitted test, so please only focus on the "Shoulds" above.

## How to submit your exercise

- Include your name in the README. If you are new to Go, please also mention this in the README so that we can consider this when reviewing your exercise
- Create a private [GitHub](https://help.github.com/en/articles/create-a-repo) repository, by copying all files you deem necessary for your submission
- [Invite](https://help.github.com/en/articles/inviting-collaborators-to-a-personal-repository) [@form3tech-interviewer-1](https://github.com/form3tech-interviewer-1) to your private repo
- Let us know you've completed the exercise using the link provided at the bottom of the email from our recruitment team

## License

Copyright 2019-2022 Form3 Financial Cloud

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
