FROM alpine:latest

COPY ./sshpot/entrypoint /entrypoint
COPY ./sshpot/createuser /bin/createuser
COPY ./build/keys/*.pub /keys/

RUN apk add --no-cache openssh asciinema curl util-linux bash \
    && ssh-keygen -A \
    && sed -i -e "s/bin\/ash/bin\/bash/" /etc/passwd \
    && sed -i 's/#PermitUserEnvironment.*/PermitUserEnvironment yes/g' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/g' /etc/ssh/sshd_config \
    && bash -c 'mkdir /root/.ssh && for i in /keys/*; do cat "$i" >> /root/.ssh/authorized_keys; done' \
    && echo "script -qea /tmp/l -T /tmp/t" >> /etc/profile

EXPOSE 22

CMD [ "/entrypoint" ]
