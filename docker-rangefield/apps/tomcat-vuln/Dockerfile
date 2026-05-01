FROM tomcat:9.0-jdk8-temurin

# 复制默认 Web 应用（manager/host-manager/ROOT）到 webapps
RUN cp -r /usr/local/tomcat/webapps.dist/* /usr/local/tomcat/webapps/

# 覆盖 tomcat-users.xml：配置弱口令 tomcat/tomcat，赋予 manager 角色
RUN cat > /usr/local/tomcat/conf/tomcat-users.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<tomcat-users xmlns="http://tomcat.apache.org/xml"
              xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
              xsi:schemaLocation="http://tomcat.apache.org/xml tomcat-users.xsd"
              version="1.0">
  <role rolename="manager-gui"/>
  <role rolename="manager-script"/>
  <role rolename="admin-gui"/>
  <user username="tomcat" password="tomcat" roles="manager-gui,manager-script,admin-gui"/>
</tomcat-users>
EOF

# 允许远程访问 Manager（替换 RemoteCIDRValve 的 allow 限制）
RUN sed -i 's|allow="[^"]*"|allow="0.0.0.0/0"|g' /usr/local/tomcat/webapps/manager/META-INF/context.xml || true
RUN sed -i 's|allow="[^"]*"|allow="0.0.0.0/0"|g' /usr/local/tomcat/webapps/host-manager/META-INF/context.xml || true

# 移除 LockOutRealm（防止 nuclei 暴力扫描时锁定用户），保留 UserDatabaseRealm
RUN perl -i -0pe 's|<Realm className="org\.apache\.catalina\.realm\.LockOutRealm">\s*<!--.*?-->\s*<Realm className="org\.apache\.catalina\.realm\.UserDatabaseRealm"\s+resourceName="UserDatabase"/>\s*</Realm>|<Realm className="org.apache.catalina.realm.UserDatabaseRealm" resourceName="UserDatabase"/>|s' /usr/local/tomcat/conf/server.xml

EXPOSE 8080
