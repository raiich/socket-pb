.PHONY: clean
clean:
	rm -rf .local
	rm -rf generated

.PHONY: generated
generated:
	$(MAKE) -C stream generated
