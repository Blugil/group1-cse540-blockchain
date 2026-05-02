FROM golang:1.21-bullseye AS build
WORKDIR /chaincode
COPY . .
RUN go build -mod=vendor -o /chaincode/bin/chaincode .

FROM hyperledger/fabric-baseos:2.5
COPY --from=build /chaincode/bin/chaincode /usr/local/bin/chaincode
EXPOSE 7052
CMD ["/usr/local/bin/chaincode"]
