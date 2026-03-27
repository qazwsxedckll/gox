.PHONY: release
release:
	standard-version --skip.changelog --skip.commit
	git push --follow-tags origin main