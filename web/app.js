import React, { useEffect, useState } from "https://esm.sh/react@18";
import { createRoot } from "https://esm.sh/react-dom@18/client";
import htm from "https://esm.sh/htm@3.1.1";

const html = htm.bind(React.createElement);
const apiBaseUrl = window.__APP_CONFIG__?.API_BASE_URL || "http://localhost:8080";

const emptyForm = {
  title: "",
  kind: "training",
  startsAt: "",
  location: "",
  maxParticipants: 16,
  notes: "",
};

const emptyChargeForm = {
  teamId: "team-pulse",
  teamName: "Pulse United",
  plan: "club",
};

function App() {
  const [activities, setActivities] = useState([]);
  const [dashboard, setDashboard] = useState(null);
  const [subscriptions, setSubscriptions] = useState([]);
  const [memberName, setMemberName] = useState("Mia");
  const [form, setForm] = useState({
    ...emptyForm,
    startsAt: new Date(Date.now() + 86400000).toISOString().slice(0, 16),
  });
  const [chargeForm, setChargeForm] = useState(emptyChargeForm);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [checkoutUrl, setCheckoutUrl] = useState("");

  async function loadData() {
    const [activitiesRes, dashboardRes, subscriptionsRes] = await Promise.all([
      fetch(`${apiBaseUrl}/api/activities`),
      fetch(`${apiBaseUrl}/api/dashboard`),
      fetch(`${apiBaseUrl}/api/subscriptions`),
    ]);

    const activitiesData = await activitiesRes.json();
    const dashboardData = await dashboardRes.json();
    const subscriptionsData = await subscriptionsRes.json();
    setActivities(activitiesData.items);
    setDashboard(dashboardData);
    setSubscriptions(subscriptionsData.items);
  }

  useEffect(() => {
    loadData().catch(() => setError("Unable to load project data."));
  }, []);

  async function createActivity(event) {
    event.preventDefault();
    setError("");
    setMessage("");
    setCheckoutUrl("");

    const response = await fetch(`${apiBaseUrl}/api/activities`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...form,
        maxParticipants: Number(form.maxParticipants),
        startsAt: new Date(form.startsAt).toISOString(),
      }),
    });

    if (!response.ok) {
      const payload = await response.json();
      setError(payload.error || "Failed to create activity.");
      return;
    }

    setForm({
      ...emptyForm,
      startsAt: new Date(Date.now() + 86400000).toISOString().slice(0, 16),
    });
    await loadData();
  }

  async function submitRSVP(activityId, status) {
    setError("");
    setMessage("");
    const response = await fetch(`${apiBaseUrl}/api/activities/${activityId}/rsvps`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ memberName, status }),
    });

    if (!response.ok) {
      const payload = await response.json();
      setError(payload.error || "Failed to save RSVP.");
      return;
    }

    await loadData();
  }

  async function startStripeCheckout(event) {
    event.preventDefault();
    setError("");
    setMessage("");

    const response = await fetch(`${apiBaseUrl}/api/checkout-sessions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...chargeForm,
        successUrl: `${window.location.origin}/billing/success`,
        cancelUrl: `${window.location.origin}/billing/cancel`,
      }),
    });

    const payload = await response.json();
    if (!response.ok) {
      setError(payload.error || "Failed to create Stripe checkout session.");
      return;
    }

    setMessage(`Stripe Checkout session created for ${payload.subscription.teamName}.`);
    setCheckoutUrl(payload.checkoutUrl || "");
    await loadData();
  }

  return html`
    <main className="shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Full-stack sports coordination demo</p>
          <h1>TeamPulse</h1>
          <p className="lede">
            A small sports coordination project for planning matches, training,
            volunteer shifts and social events with a Go API and React UI.
          </p>
        </div>
        <div className="hero-card">
          <label>RSVP as</label>
          <input
            value=${memberName}
            onChange=${(event) => setMemberName(event.target.value)}
            placeholder="Member name"
          />
          <p>
            Use the cards below to record attendance and watch the dashboard
            update.
          </p>
        </div>
      </section>

      ${dashboard &&
      html`
        <section className="stats">
          <article>
            <span>Upcoming</span>
            <strong>${dashboard.upcomingActivities}</strong>
          </article>
          <article>
            <span>Total going</span>
            <strong>${dashboard.totalGoing}</strong>
          </article>
          <article>
            <span>Next event</span>
            <strong>${dashboard.nextActivity ? dashboard.nextActivity.title : "None"}</strong>
          </article>
          <article>
            <span>Subscriptions</span>
            <strong>${subscriptions.length}</strong>
          </article>
        </section>
      `}

      <section className="grid">
        <div className="stack">
          <form className="panel" onSubmit=${createActivity}>
            <div className="panel-heading">
              <h2>Create activity</h2>
              <p>Demonstrates JSON form submission into the activity service.</p>
            </div>

            <label>
              Title
              <input
                value=${form.title}
                onChange=${(event) => setForm({ ...form, title: event.target.value })}
                required
              />
            </label>

            <label>
              Type
              <select
                value=${form.kind}
                onChange=${(event) => setForm({ ...form, kind: event.target.value })}
              >
                <option value="match">Match</option>
                <option value="training">Training</option>
                <option value="volunteer">Volunteer</option>
                <option value="social">Social</option>
              </select>
            </label>

            <label>
              Starts at
              <input
                type="datetime-local"
                value=${form.startsAt}
                onChange=${(event) => setForm({ ...form, startsAt: event.target.value })}
                required
              />
            </label>

            <label>
              Location
              <input
                value=${form.location}
                onChange=${(event) => setForm({ ...form, location: event.target.value })}
                required
              />
            </label>

            <label>
              Max participants
              <input
                type="number"
                min="1"
                value=${form.maxParticipants}
                onChange=${(event) =>
                  setForm({ ...form, maxParticipants: event.target.value })
                }
                required
              />
            </label>

            <label>
              Notes
              <textarea
                value=${form.notes}
                onChange=${(event) => setForm({ ...form, notes: event.target.value })}
              ></textarea>
            </label>

            <button type="submit">Create activity</button>
          </form>

          <form className="panel" onSubmit=${startStripeCheckout}>
            <div className="panel-heading">
              <h2>Start Stripe checkout</h2>
              <p>Creates a Stripe subscription checkout session in the payment service.</p>
            </div>

            <label>
              Team ID
              <input
                value=${chargeForm.teamId}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, teamId: event.target.value })
                }
                required
              />
            </label>

            <label>
              Team name
              <input
                value=${chargeForm.teamName}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, teamName: event.target.value })
                }
                required
              />
            </label>

            <label>
              Plan
              <select
                value=${chargeForm.plan}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, plan: event.target.value })
                }
              >
                <option value="starter">Starter</option>
                <option value="club">Club</option>
                <option value="pro">Pro</option>
              </select>
            </label>

            <button type="submit">Create checkout session</button>
          </form>

          ${(error || message) &&
          html`
            <section className="panel feedback">
              ${error && html`<p className="error">${error}</p>`}
              ${message && html`<p className="success">${message}</p>`}
              ${checkoutUrl &&
              html`<p className="success">Checkout URL: <a href=${checkoutUrl}>${checkoutUrl}</a></p>`}
            </section>
          `}
        </div>

        <section className="panel">
          <div className="panel-heading">
            <h2>Upcoming activities</h2>
            <p>Each card supports quick RSVP interactions.</p>
          </div>

          <div className="activity-list">
            ${activities.map(
              (activity) => html`
                <article className="activity-card" key=${activity.id}>
                  <div className="activity-meta">
                    <span className="kind">${activity.kind}</span>
                    <span>
                      ${new Date(activity.startsAt).toLocaleString([], {
                        dateStyle: "medium",
                        timeStyle: "short",
                      })}
                    </span>
                  </div>
                  <h3>${activity.title}</h3>
                  <p>${activity.location}</p>
                  <p className="notes">${activity.notes}</p>
                  <div className="attendance">
                    <span>Going ${activity.goingCount}</span>
                    <span>Maybe ${activity.maybeCount}</span>
                    <span>Declined ${activity.declinedCount}</span>
                  </div>
                  <div className="actions">
                    <button onClick=${() => submitRSVP(activity.id, "going")}>Going</button>
                    <button onClick=${() => submitRSVP(activity.id, "maybe")}>Maybe</button>
                    <button onClick=${() => submitRSVP(activity.id, "declined")}>
                      Decline
                    </button>
                  </div>
                </article>
              `
            )}
          </div>

          <div className="panel-heading subscriptions-heading">
            <h2>Subscriptions</h2>
            <p>Billing state is isolated in the payment microservice.</p>
          </div>
          <div className="activity-list">
            ${subscriptions.map(
              (subscription) => html`
                <article className="activity-card" key=${subscription.teamId}>
                  <div className="activity-meta">
                    <span className="kind">${subscription.plan}</span>
                    <span>${subscription.status}</span>
                  </div>
                  <h3>${subscription.teamName}</h3>
                  <p>${subscription.teamId}</p>
                  <p className="notes">
                    NOK ${subscription.priceMonthlyNOK}/month, renews
                    ${new Date(subscription.renewalDate).toLocaleDateString()}.
                  </p>
                  <div className="attendance">
                    <span>Last payment ${subscription.lastPaymentStatus}</span>
                    ${subscription.stripeSessionId &&
                    html`<span>Stripe session ${subscription.stripeSessionId}</span>`}
                  </div>
                </article>
              `
            )}
          </div>
        </section>
      </section>
    </main>
  `;
}

createRoot(document.getElementById("root")).render(html`<${App} />`);
