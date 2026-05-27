FROM ethereum/client-go:v1.13.15

COPY genesis.json /genesis.json
COPY password.txt /password.txt
COPY accounts /accounts
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 8545

ENTRYPOINT ["/entrypoint.sh"]
