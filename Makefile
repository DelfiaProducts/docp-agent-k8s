# build applications
#REPOSITORY -> env for repository name
#TAG -> env for image tag
#PLATFORM -> env for build platform - linux/arm64, linux/amd64
#DOCKERFILE -> env for Dockerfile path
build:
	@echo "Building application..."
	@echo "Repository - ${REPOSITORY}"
	@echo "Tag - ${TAG}"
	@echo "Platform - ${PLATFORM}"
	@echo "Dockerfile - ${DOCKERFILE}"
	@docker build -t ${REPOSITORY}:${TAG} --platform ${PLATFORM} -f ${DOCKERFILE} .
