# How to Contribute

## Join the Community

The best way to get involved is to join the [CHAOSS Slack](https://chaoss.community/kb-getting-started/) in the #data-ai-disclosure-detection channel to be part of the conversation.

These resources are a great way to meet the people behind the project, ask questions, get help, participate in discussions, and stay updated on community meetings and planning. Everyone is welcome, so feel free to introduce yourself and ask for help if you get stuck or feel frustrated with any part of this setup process!

## Opening an issue
If you're experiencing an issue you can search for your problem or question on our [issues](https://github.com/chaoss/disclosure/issues) page to see if someone else has already reported it. If you cannot find your issue, please feel free to [open a new one](https://github.com/chaoss/disclosure/issues/new/choose).

If you are new to opening issues, we recommend [opensource.guide](https://opensource.guide/how-to-contribute) and their section on [Opening Issues](https://opensource.guide/how-to-contribute/#opening-an-issue).

> [!TIP]
> Filling our our issue templates will help us gather all the necessary information to troubleshoot your issue efficiently. Issues that are missing details may take longer to be fixed.

## Contribution Standards

In order to maintain a high level of code quality, we have certain standards of behavior and contribution quality we expect from all contributors and contributions (regardless of whether those changes were made by hand or with tools).

- **Discuss first, then code.** Open an issue, leave a comment, or reach out on Slack before submitting a PR (especially if its your first time). This creates an opportunity to get early feedback on your plan from the maintainers and allows you to check if the task has been claimed by someone else, avoiding wasted effort on both sides.
- **Maintain situational awareness of your code and take responsibility.** The power to make changes to code that other people use requires taking responsibility. As a rule of thumb, if you can't explain why your change works or aren't prepared to answer follow-up questions from maintainers during review, consider picking a different issue to solve or asking maintainers for resources to help you learn the relevant concepts or details about the codebase.
- **Be honest and upfront.** Honesty, integrity, and trust are an important part of how we maintain the quality of our codebase. We require that the sources be cited when they are not written by the author of the PR. This includes, but is not limited to: citing Stack Overflow when you copy and paste code, correctly attributing co-authorship on commits that are made collaboratively, and completing the AI disclosure and DCO portions of the PR template truthfully.
- **Respect others time.** Project maintainers and other contributors may be volunteering their time to review your work, please be courteous and respect their time. If you can't respond to something for a few days, just say so! If your proposed changes (or any comments about them) are too long for you to proofread, they're likely too long for maintainers to read and should be reorganized or made smaller if possible.
- **Remember the human.** Remember that the humans who are helping build this project are on the other side of your issues, comments, and Slack messages. Your words and actions should remain respectful, even if you disagree. Using inflammatory or hateful language is not acceptable. Simply outsourcing your workload to an LLM and relaying the output back doesn't help anyone grow and doesn't sustain our community.


Following these standards when you contribute will help your contributions get merged faster. If a maintainer determines your change to be in violation of one of these standards, you will be informed and given an opportunity to correct your PR.

Repeat violations of these standards may result in the closure of your PRs, a ban from the project, referral for [CHAOSS Code of Conduct](https://www.chaoss.community/code-of-conduct/) violations, or other remedies deemed by the maintainers to be in the best interests of the overall health of the project. 

## How to contribute to the source code
We welcome pull requests from anyone!

We follow the same GitHub workflow that most other projects on GitHub follow: Fork -> create a branch -> make a pull request -> repeat.

Detailed instructions for making your contribution under this workflow can be found on the [GitHub Flow page](https://docs.github.com/en/get-started/using-github/github-flow). There is also an opensource.guide section on [making pull requests](https://opensource.guide/how-to-contribute/#opening-a-pull-request). If you get stuck, please ask for help in the project Slack.

### Signing-off on Commits
To contribute to this project, you must agree to the [Developer Certificate of Origin](https://developercertificate.org/) (DCO) for each commit you make. The DCO is a simple statement that you, as a contributor, have the legal right to make the contribution. It is NOT a copyright assignment or transfer. This certification is required for contributions to CHAOSS repositories by the [CHAOSS charter](https://chaoss.community/about/charter/#user-content-8-intellectual-property-policy).

To signify that you agree to the DCO for contributions, you simply add a line to each of your git commit messages. For example:
```
Signed-off-by: Jane Smith <jane.smith@example.com>
```

This can be easily done by using the `-s` flag when running the `git commit` command: `git commit -s -m "my commit message"`

The PiHole project has more detailed guide on adding this signoff to your commits can be found on their ["How to sign-off commits"](https://docs.pi-hole.net/guides/github/how-to-signoff/) page.

> [!TIP]
> Signing off commits is slightly easier and safer if you do it before you push your changes to GitHub.


_The contents of this contributing.md were adapted from the [CollectOSS CONTRIBUTING guidelines](https://github.com/chaoss/CollectOSS/blob/61f5765f9b7e8cd633454e618f8714d111078dbd/CONTRIBUTING.md)_
