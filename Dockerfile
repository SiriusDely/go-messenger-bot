FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD messenger-bot index.html /
EXPOSE 8080
CMD ["/messenger-bot"]
