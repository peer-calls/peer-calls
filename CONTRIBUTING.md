# Contributing

Thank you for your interest to contribute to Peer Calls. By participating in
this project, you agree to abide by the [code of conduct].

  [code of conduct]: CODE_OF_CONDUCT.md

Everyone is expected to follow the code of conduct anywhere in the project
codebases, issue trackers and chatrooms.

## Notice

Before contributing, check the issue tracker and see if there is already an
issue about something you're trying to implement. Check if somebody else is
already assigned to the issue. It is recommended to *ask* or create an issue
before implementing changes - this way you'll have a lower chance that your
pull request will be rejected.

## Rules

 1. Do not include libraries or fonts from a CDN. All libraries must be bundled
    with the app, preferrably installed via NPM.
 2. Do not use APIs from third-party servers, like Google Maps or Slack.
 3. If a permission is needed, do not ask for it before explaining the need for
    it to the user. From [Google Web Fundamentals][gwf]:
    > Make sure that users understand why you’re asking for their location, and
    > what the benefit to them will be. Asking for it immediately on the
    > homepage as the site loads results in a poor user experience.
    One exception is the camera/microphone access, because this application is
    useless without it.
 4. Write tests! A lot of work has been put to have a high test code coverage,
    and pull requests that decrease test coverage will be rejected.
 5. Do not include your personal development environment settings in this
    repository. Different people use different environments, and only generic
    settings like `.editorconfig` will be allowed.
 6. In order to accept a pull request, the CI must pass, and that includes
    the linter, all tests, and the build process.

  [gwf]: https://developers.google.com/web/fundamentals/native-hardware/user-location/

## Contributing Code

1. Fork the repo
2. Install dependencies: `npm install`
3. Make sure the tests pass: `npm run ci`
4. Make your change on a new feature branch, with new passing tests. We use
   ESLint to lint the code.
5. Push to your fork.
6. Make sure to rebase to master and squash commits to a single commit.
7. Write a [good commit message][commit].
   - [Reference][reference] the issue in the commit message.
   - Remember that you can also [close][close] the issue with this message.
8. Submit a pull request.

  [commit]: http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html
  [reference]: https://help.github.com/articles/autolinked-references-and-urls/#issues-and-pull-requests
  [close]: https://help.github.com/articles/closing-issues-via-commit-messages/

Others will give constructive feedback.  This is a time for discussion and
improvements, and making the necessary changes will be required before we can
merge the contribution.

## License

As the project uses the [MIT license][license], you agree that any code you submit to the
project will also be licensed under the same license.

[license]: LICENSE
