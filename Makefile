OUTPUT_DIR=extension-root
FUNCTION_NAME=extension-test
EXTENSION_BIN=sq-lambda-secrets-prefetch
ARN:= $(shell cat .layer.arn.txt)
RESPONSEFILE=response.txt
LAMBDA_FUNCTION=example-lambda
FUNCTION_BIN=main.py
FUNCTION_OUTPUT_DIR=build/example-lambda
LAYER_NAME=extension-test-layer
EXECUTION_ROLE=arn:aws:iam::1234567890:role/extension-test-role 


clean:
	rm -f extensions.zip
	rm -rf "${FUNCTION_OUTPUT_DIR}"
	rm -rf build/*.zip

build: build-extension build-function ;

build-extension:
	mkdir -p build
	mkdir -p "${OUTPUT_DIR}"
	GOOS=linux go build -o "${OUTPUT_DIR}/extensions/${EXTENSION_BIN}" extension/extension.go
	cd "${OUTPUT_DIR}" && zip -r ../build/extensions.zip . -x *.DS_Store *.gitkeep

build-function:
	mkdir -p build
	mkdir -p "${FUNCTION_OUTPUT_DIR}"
	cp "${LAMBDA_FUNCTION}/main.py" "${FUNCTION_OUTPUT_DIR}"
	cp "${LAMBDA_FUNCTION}/config.yaml" "${FUNCTION_OUTPUT_DIR}"
	cd "${FUNCTION_OUTPUT_DIR}" && zip -j -r ../function.zip "${FUNCTION_BIN}" "config.yaml"


update-layer: check-region-profile-set build-extension
	aws lambda publish-layer-version \
		--layer-name "${LAYER_NAME}" \
		--compatible-runtimes go1.x ruby2.7 python3.7 \
		--region "${AWS_REGION}" \
		--zip-file  "fileb://build/extensions.zip" > .response.json
	cat .response.json | jq  -r '.LayerVersionArn' > .layer.arn.txt
	@echo 
	@echo "Layer ARN"
	@echo 
	cat .layer.arn.txt

.PHONY: run
run: check-region-profile-set
	aws lambda invoke \
		--function-name "${FUNCTION_NAME}" \
		--region "us-east-1" \
		--invocation-type "Event" \
		${RESPONSEFILE}

delete-function: check-region-profile-set
	aws lambda delete-function \
		--region "${AWS_REGION}" \
		--function-name "${FUNCTION_NAME}"

create-function: check-region-profile-set check-arn-set build-function
	aws lambda create-function \
		--function-name "${FUNCTION_NAME}" \
		--runtime "python3.7" \
		--role "${EXECUTION_ROLE}" \
		--layers ${ARN} \
		--region "${AWS_REGION}" \
		--handler "main.my_handler" \
		--zip-file "fileb://build/function.zip"

update-function: check-region-profile-set check-arn-set build-function
	aws lambda update-function-code \
		--function-name "${FUNCTION_NAME}" \
		--zip-file "fileb://build/function.zip" \
		--region ${AWS_REGION}
	aws lambda update-function-configuration \
		--function-name "${FUNCTION_NAME}" \
		--region "${AWS_REGION}" \
		--layers "${ARN}"
	aws lambda publish-version \
		--function-name "${FUNCTION_NAME}" \
		--description "extension test" \
		--region ${AWS_REGION}

check-region-profile-set:
	if [ -z "${AWS_REGION}" ]; then \
        echo "AWS_REGION environment variable is not set."; exit 1; \
    fi
	if [ -z "${AWS_PROFILE}" ]; then \
        echo "AWS_PROFILE environment variable is not set."; exit 1; \
    fi

check-arn-set:
	if [ -z "${ARN}" ]; then \
        echo "File .layer.arn.txt not set."; exit 1; \
    fi


test: build
	go test ./...
