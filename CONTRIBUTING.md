# Contributing to Ciao

Ciao is an open source project licensed under the [Apache v2 License] (https://opensource.org/licenses/Apache-2.0)

## Coding Style

Ciao uses the golang coding style, go fmt is your friend.

## Certificate of Origin

In order to get a clear contribution chain of trust we use the [signed-off-by language] (https://01.org/community/signed-process)
used by the Linux kernel project.

## Patch format

Beside the signed-off-by footer, we expect each patch to comply with the following format:

```
       <component>: Change summary

       More detailled explanation of your changes: Why and how.
       Wrap it to 72 characters.
       See [here] (http://chris.beams.io/posts/git-commit/)
       for some more good advices.

       Signed-off-by: <contributor@foo.com>
```

For example:

```
	ssntp: Implement role checking

	SSNTP roles are contained within the SSNTP certificates
	as key extended attributes. On both the server and client
	sides we are verifying that the claimed roles through the
	SSNTP connection protocol match the certificates.

	Signed-off-by: Samuel Ortiz <sameo@linux.intel.com>
```

## Pull requests

We accept github pull requests.

If you want to work on github.com/01org/ciao and your fork on the same workstation you will need to use multiple GOPATHs.  Assuming this is the case

1. Open a terminal
2. mkdir -p ~/go-fork/src/github.com/01org (replacing go-fork with your preferred location)
3. export GOPATH=~/go-fork
4. cd $GOPATH/src/github.com/01org
5. git clone https://github.com/\<github-username\>/ciao.git (replace \<github-username\> with your username)
6. cd ciao
7. go install ./...

Once you've finished making your changes push them to your fork and send the PR via the github UI.  If you don't need to maintain the github.com/01org/ciao repo and your fork on the same workstation you can skip steps 2 and 3.
