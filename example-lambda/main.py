#!/usr/bin/env python3

import os
import boto3
import base64
from botocore.exceptions import ClientError

# This lambda only reads the /tmp/secrets directory and logs the contents of all the files there.
def my_handler(event, context):
    secrethome = "/tmp/secrets"

    print("Listing: {}".format(secrethome))

    print(os.listdir(secrethome))
    for x in os.listdir(secrethome):
        print ("Reading: {}/{}".format(secrethome, x))
        with open("{}/{}".format(secrethome, x)) as f:
            print(f.readlines())
            print()

if __name__ == "__main__":
    my_handler(None, None)


