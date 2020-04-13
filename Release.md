1. Export `$VERSION`:

      export VERSION=0.10.0

2. Add new version to file VERSION:

      echo "${VERSION}" | tee VERSION && git commit -m "Update VERSION file for ${VERSION}" VERSION

3. Move changelog files for `calens`:

      mv changelog/unreleased "changelog/${VERSION}_$(date +%Y-%m-%d)"
      git add "changelog/${VERSION}"*
      git rm -r changelog/unreleased
      git commit -m "Move changelog files for ${VERSION}" changelog/{unreleased,"${VERSION}"*}

4. Generate changelog:

      calens > CHANGELOG.md
      git add CHANGELOG.md
      git commit -m "Generate CHANGELOG.md for ${VERSION}" CHANGELOG.md

5. Tag new version and push the tag:

      git tag -a -s -m "v${VERSION}" "v${VERSION}"
      git push --tags

6. Build the project (use `--skip-publish` for testing):

      goreleaser \
        release \
        --config ../.goreleaser.yml \
        --release-notes <(calens --template changelog/CHANGELOG-GitHub.tmpl --version "${VERSION}")

