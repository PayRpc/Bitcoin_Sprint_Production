# Grafana Dockerfile for Fly.io
FROM grafana/grafana:10.2.0

# Create necessary directories
USER root
RUN mkdir -p /var/lib/grafana/dashboards /etc/grafana/provisioning/datasources /etc/grafana/provisioning/dashboards

# Copy provisioning files
COPY grafana/provisioning/datasources/prometheus.yml /etc/grafana/provisioning/datasources/
COPY grafana/provisioning/dashboards/dashboard.yml /etc/grafana/provisioning/dashboards/

# Copy dashboard files
COPY grafana/dashboards/ /var/lib/grafana/dashboards/

# Set proper permissions
RUN chown -R grafana:grafana /var/lib/grafana /etc/grafana

# Switch back to grafana user
USER grafana

# Environment variables for Fly.io
ENV GF_SERVER_HTTP_PORT=8080
ENV GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
ENV GF_USERS_ALLOW_SIGN_UP=false
ENV GF_INSTALL_PLUGINS=grafana-piechart-panel,grafana-worldmap-panel

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Expose port
EXPOSE 8080
