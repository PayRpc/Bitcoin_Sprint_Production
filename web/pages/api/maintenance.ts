import {
    createMaintenanceMode,
    disableMaintenanceMode,
    getMaintenanceStatus,
    updateSystemState
} from "@/lib/maintenance";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, DELETE");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  try {
    switch (req.method) {
      case "GET":
        // Get maintenance status
        const status = await getMaintenanceStatus();
        return res.status(200).json({ ok: true, maintenance: status });

      case "POST":
        // Enable maintenance mode or update system state
        const { action, reason, version, rollback } = req.body;

        if (action === "enable_maintenance") {
          await createMaintenanceMode(reason);
          return res.status(200).json({ 
            ok: true, 
            message: "Maintenance mode enabled",
            maintenance: { enabled: true, reason }
          });
        }

        if (action === "update_state") {
          if (!version) {
            return res.status(400).json({ ok: false, error: "Version is required" });
          }
          await updateSystemState(version, rollback || false);
          return res.status(200).json({ 
            ok: true, 
            message: "System state updated",
            version,
            rollback: rollback || false
          });
        }

        return res.status(400).json({ ok: false, error: "Invalid action" });

      case "DELETE":
        // Disable maintenance mode
        await disableMaintenanceMode();
        return res.status(200).json({ 
          ok: true, 
          message: "Maintenance mode disabled",
          maintenance: { enabled: false }
        });

      default:
        return res.status(405).json({ ok: false, error: "Method not allowed" });
    }
  } catch (e: any) {
    return res.status(500).json({
      ok: false,
      error: e.message || "Failed to manage maintenance",
    });
  }
}
