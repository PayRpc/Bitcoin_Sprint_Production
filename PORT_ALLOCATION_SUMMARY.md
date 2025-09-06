# Port Configuration Summary - Bitcoin Sprint
# Generated: September 5, 2025
# Status: All conflicts resolved ✅

## Port Allocation by Tier:

### FREE TIER (.env.free):
- API_PORT:    8080
- WEB_PORT:    3000  
- ADMIN_PORT:  8081

### BUSINESS TIER (.env.business):
- API_PORT:    8082
- WEB_PORT:    3001
- ADMIN_PORT:  8083

### ENTERPRISE TIER (.env.enterprise):
- API_PORT:    9000
- WEB_PORT:    3002
- ADMIN_PORT:  9001

## Service Ports:
- service-config.toml:    8090 (Updated from 8080)
- enterprise-service.toml: 8443 (No conflict)
- Prometheus:             9090 (Monitoring)
- Grafana:                3003 (Production monitoring - Updated from 3000)

## Fixed Conflicts:
1. ✅ Enterprise API/ADMIN port conflict (both were 9000)
2. ✅ Service config port overlap with FREE tier
3. ✅ Tier separation for web dashboard ports
4. ✅ Grafana moved to port 3003 for production monitoring

## Production Status:
- ✅ Grafana (3003) - Production monitoring always available
- ✅ FREE web (3000) - No conflicts with monitoring infrastructure
- ✅ All blockchain node ports remain consistent (8333, 28332, 8335)

## Validation:
- ✅ No ports currently in use
- ✅ All tiers have unique port ranges
- ✅ Admin, API, and Web ports separated per tier
