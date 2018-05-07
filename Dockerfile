FROM centos:7.3.1611
ENV BaseUrl https://github.com/happy2048/angelina2/blob/master
RUN yum install epel-release -y
RUN yum install wget git -y
EXPOSE 6300
RUN wget -c  $BaseUrl/bin/angelina-controller -O /usr/bin/angelina-controller  && \
	wget -c $BaseUrl/bin/angelina-runner   -O /root/angelina-runner && \
	wget -c $BaseUrl/utils/redis-cli   -O /usr/bin/redis-cli  && \
	wget -c $BaseUrl/utils/socket_client -O /usr/bin/socket_client
RUN chmod +x /usr/bin/angelina-controller && \
	chmod +x /usr/bin/redis-cli && \
	chmod +x /usr/bin/socket_client
ENTRYPOINT ["angelina-controller"]
