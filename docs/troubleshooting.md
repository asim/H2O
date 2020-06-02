You can find some errors & solutions faced while trying to get h2 up and working.

#### If you have issues about certificates when you want to run `docker/build.sh` 

Re-try after running `eval "$(docker-machine env h2)"`. Docker needs certificates for the communication between the host and client.

#### If after login you don't get redirected back to a dashboard

It might be that the docker host's time is hours behind the correct time. Fix it by 

```
docker-machine ssh h2
sudo ntpclient -s -h pool.ntp.org
```

#### If you get `could not read Username for 'https://github.com': terminal prompts disabled` when you run `go get`

There might be 2 reasons for this. 

First, your Git configuration might be wrong. Be sure you set your username and email and ssh key correctly and you are able to
ssh into Github.

Second, Although your git config is correct, your permission might be wrong. Ask someone in the team for the permissions.

After all, if you are still getting the error try cloning the repository manually via SSH and running `go get` again. As it will see the folder in the correct place, it will not try to
fetch it from Github over HTTP and you will not get the error. This error is something related to HTTP authentication of Github but I don't know what's wrong with it.