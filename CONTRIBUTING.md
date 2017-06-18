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
