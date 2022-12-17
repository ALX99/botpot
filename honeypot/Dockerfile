FROM alpine:latest

COPY ./honeypot/entrypoint /entrypoint
COPY ./honeypot/createuser /bin/createuser

RUN apk add --no-cache openssh asciinema curl util-linux bash \
  && ssh-keygen -A \
  && sed -i -e "s/bin\/ash/bin\/bash/" /etc/passwd \
  && sed -i 's/#PermitUserEnvironment.*/PermitUserEnvironment yes/g' /etc/ssh/sshd_config \
  && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication yes/g' /etc/ssh/sshd_config \
  && sed -i 's/#PermitEmptyPasswords.*/PermitEmptyPasswords yes/g' /etc/ssh/sshd_config \
  && sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/g' /etc/ssh/sshd_config \
  && sed -i 's/#PermitTunnel.*/PermitTunnel yes/g' /etc/ssh/sshd_config \
  && sed -i 's/AllowTcpForwarding.*/AllowTcpForwarding yes/g' /etc/ssh/sshd_config \
  && echo 'PermitOpen any' >>/etc/ssh/sshd_config \
  && passwd -d root \
  && echo "script -qea /tmp/l -T /tmp/t" >> /etc/profile

EXPOSE 22

CMD [ "/entrypoint" ]
