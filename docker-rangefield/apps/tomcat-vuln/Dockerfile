FROM tomcat:9.0-jdk8-temurin

# 开启 Tomcat Manager / Host-Manager，配置弱口令 tomcat/tomcat
RUN sed -i 's|<\/tomcat-users>|<role rolename="manager-gui"\/>\n<role rolename="manager-script"\/>\n<role rolename="admin-gui"\/>\n<user username="tomcat" password="tomcat" roles="manager-gui,manager-script,admin-gui"\/>\n<\/tomcat-users>|' /usr/local/tomcat/conf/tomcat-users.xml

# 允许远程访问 Manager（默认只允许 127.0.0.1）
RUN sed -i 's|allow="127\\.\\d+\\.\\d+\\.\\d+|::1|0:0:0:0:0:0:0:1"|allow=".*"|g' /usr/local/tomcat/webapps/manager/META-INF/context.xml || true
RUN sed -i 's|allow="127\\.\\d+\\.\\d+\\.\\d+|::1|0:0:0:0:0:0:0:1"|allow=".*"|g' /usr/local/tomcat/webapps/host-manager/META-INF/context.xml || true

EXPOSE 8080
